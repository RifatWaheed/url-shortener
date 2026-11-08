# Week 4 Notes — Caching (Redis) + Go details from the build

---

## 1. The cache layer we built

**Q: What caching strategy does `CachedStore` use?**
Ans: Cache-aside (lazy loading). On a redirect, check Redis first. On a hit, return the cached link. On a miss, read from Postgres, write the result into Redis with a TTL, then return it. The cache is only filled when someone actually asks for a code, so we never cache links nobody uses.

Follow-up: *Why cache-aside and not write-through?*
Ans: Write-through puts every new link into the cache when it's created, so writes get slower and the cache fills with links that may never be clicked. A shortener is very read-heavy, with one create and many redirects, so loading lazily on the first read fits better. The cost is that the first redirect of each link is always a miss.

**Q: How is `CachedStore` wired around the Postgres store?**
Ans: It embeds the `LinkStore` interface and adds a Redis client and a TTL. Because of the embedding, every method we don't override (like `CreateLink`) goes straight to the real store. We only override `GetLink`. This is the decorator pattern: the handlers still depend on `LinkStore` and don't know whether a cache is there.

Follow-up: *Why does that matter?*
Ans: The handlers and their tests don't change. We can turn caching on or off in `main.go` by choosing which store to pass in, and we can test the cache in isolation by wrapping a fake store.

**Q: Why prefix the key as `link:<code>`?**
Ans: It namespaces keys in Redis. Later the same Redis will hold rate-limit counters (Week 6) and maybe negative-cache markers. Prefixes stop keys from colliding and let you scan or delete one kind of key at a time.

**Q: What happens if Redis is down?**
Ans: `Get` returns an error that isn't `redis.Nil`. We log a warning and fall through to Postgres, so the redirect still works, just slower. This is "failing open": the cache is an optimization, not something we depend on to be correct. A failed `Set` is also only logged, never returned to the user.

Follow-up: *Risk of failing open?*
Ans: If Redis dies under heavy traffic, all of that load lands on Postgres at once. Postgres needs to be sized for that, or protected with rate limiting or a circuit breaker.

---

## 2. TTL and link expiry

**Q: How do you choose the TTL for a cached link?**
Ans: Use the default (24h, `24 * time.Hour`) unless the link expires sooner. Then the TTL is the time left until expiry, `min(c.ttl, time.Until(*ExpiresAt))`. That way the cache entry disappears at the same moment the link stops being valid, and the cache can never serve an expired link.

**Q: Why can't you compare `ExpiresAt` with `c.ttl` directly?**
Ans: They're different types. `ExpiresAt` is a `*time.Time`, a point on the clock. `ttl` is a `time.Duration`, a length of time. Go won't compare a moment with a length. `time.Until(t)` converts the moment into "how long from now until then", which is a `Duration`, and then the two can be compared.

**Q: What if the link expires between the DB read and the cache write?**
Ans: The repository said the link was valid, but a few milliseconds later `time.Until` comes out zero or negative. We return the link (it was valid when it was read) but skip caching it.

Follow-up: *Why not cache it anyway with that TTL?*
Ans: In go-redis, `Set` with an expiration of 0 means **no expiry**. Caching with zero or negative time would keep a dead link in Redis forever. An edge-case race would become a permanent bug.

**Q: Why put the TTL default in the struct instead of hardcoding 24h in `GetLink`?**
Ans: `GetLink` shouldn't know fixed numbers. Setting `ttl` where the `CachedStore` is built means it can later come from config or env vars without changing the lookup logic, and tests can use a short TTL.

---

## 3. Negative caching and cache penetration

**Q: What is cache penetration?**
Ans: Requests for keys that don't exist anywhere. They miss the cache every time, so every one goes to the database. An attacker sending made-up short codes can use this to hammer Postgres even though the cache is in place.

**Q: What is negative caching?**
Ans: Caching the "not found" answer. When Postgres returns `ErrLinkNotFound`, store a sentinel marker for that key with a **short** TTL. The next lookup of the same bad code gets "not found" from Redis without touching the DB. On a hit, check for the marker *before* calling `json.Unmarshal`.

Follow-up: *Why a short TTL for negative entries?*
Ans: A code that doesn't exist now might be created soon. A long negative TTL would make a brand-new link return 404 until it ran out.

Follow-up: *Which errors should you negatively cache?*
Ans: Only a real `ErrLinkNotFound`, checked with `errors.Is`. Never cache infrastructure errors. If Postgres is briefly down and we cache "not found", real links return 404 for the whole TTL, and a short DB blip becomes a much longer outage.

**Q: Does negative caching stop an attacker hammering random codes?**
Ans: No. It only helps when the **same** bad code is requested again and again. An attacker can make a fresh random code for every request. Each one is still a first-time miss that reaches Postgres, and now each one also writes a junk key into Redis, so the attack also costs memory.

**Q: So what actually protects against random-code hammering?**
Ans: Layers, cheapest first:
1. **Input validation.** Reject malformed codes (empty, too long, bad characters) before touching Redis or Postgres. Already built in `validations.go`.
2. **Rate limiting per client.** Cap how many lookups one IP or key can force. This is the main defense against a single abusive source (Week 6, token bucket in Redis).
3. **Negative cache.** Absorbs repeats of the same bad code.
4. **Bloom filter.** Answers "definitely doesn't exist" from memory for *any* code, even ones never seen before.

**Q: What is a Bloom filter, and what are its guarantees?**
Ans: A compact bit array plus several hash functions. Adding a key sets a few bits. Checking a key looks at those bits. If any bit is 0, the key is **definitely not** present. If all are 1, it's **maybe** present, because other keys could have set those bits. So it can give false positives, which just means a wasted DB lookup, but never false negatives.

**Q: Did we add a Bloom filter? Why not?**
Ans: No. At this scale, validation, rate limiting and negative caching are enough, and a Bloom filter has real costs:
- **Keeping it in sync.** Every create must add the code to the filter. If that write fails, or the filter is lost on restart, real links get false negatives and return 404, which is worse than the attack.
- **No deletes.** A standard Bloom filter can't remove entries, so expired links stay "maybe present" and the false-positive rate creeps up. You'd need rebuilds or a counting/cuckoo filter.
- **Infrastructure.** `BF.ADD` / `BF.EXISTS` need the RedisBloom module (Redis Stack), not plain Redis.

Follow-up: *When does it become the right call?*
Ans: When the attack is **distributed**. A botnet across thousands of IPs gets around per-IP rate limits, and random codes get around the negative cache. At hundreds of millions of links, the cost of keeping the filter in sync is worth paying.

**Interview summary: "How do you handle lookups for keys that don't exist?"**
- Validate the input first, then negatively cache genuine not-found results with a short TTL.
- Never cache infrastructure errors as "not found".
- On create, invalidate any negative entry for that key.
- Rate limit per client against a single abusive source. Add a Bloom filter in front of the DB when misses are distributed and the keyspace is huge.

---

## 4. Thundering herd / cache stampede

**Q: What is a cache stampede?**
Ans: A popular key expires, and many requests miss at the same moment. They all query Postgres for the same row and all write it back to Redis. One expiry turns into hundreds of identical DB queries.

**Q: How do you prevent it?**
Ans:
- **Request coalescing.** Only one request goes to the DB for a given key, and the rest wait for its result. In Go this is `golang.org/x/sync/singleflight`, which works inside one process.
- **Distributed lock.** `SET lock:<code> NX PX <ms>` in Redis, so only one instance across the fleet rebuilds the key.
- **TTL jitter.** Add a small random amount to each TTL so keys cached at the same time don't all expire together.
- **Early refresh.** Refresh hot keys a little before they expire.

---

## 5. Go fundamentals that came up while building

**Q: Explain pointers and dereferencing.**
Ans: A pointer holds the **address** of a variable, not its value. `&x` means "the address of x". `*p` means "go to the address in p and use the value there". Writing `*p = 20` changes `x`, because both refer to the same memory. In a type, `*Link` means "pointer to a Link". In an expression, `*p` means "dereference p". It's the same symbol in two roles.

**Q: Why is `ExpiresAt` a `*time.Time` instead of a `time.Time`?**
Ans: A pointer can be `nil`, and a plain `time.Time` can't. `nil` represents "never expires", which matches `NULL` in the `expires_at` column. That's why we check `!= nil` before using `*ExpiresAt`. Dereferencing a nil pointer has no address to follow, so Go panics with `nil pointer dereference`.

**Q: Why do `json.Unmarshal` and `Scan` need `&`?**
Ans: Go passes everything by value. Passing `link` would give the function a copy. It would fill the copy, and our variable would stay empty. Passing `&link` gives it the address, so it writes into our variable.

**Q: What caused `cannot use &link (value of type **Link) as Link`?**
Ans: `link` was declared as `*Link`, so `&link` was a pointer to a pointer (`**Link`). The function returns a plain `Link`. The fix is to declare `var link Link`, unmarshal into `&link`, and return `link`. Each `&` adds a pointer layer and each `*` removes one, so pick one representation and keep to it.

**Q: When should you use a pointer vs a value?**
Ans: Use a pointer when a function must change the caller's variable (`Unmarshal`, `Scan`), when "no value" must be possible (`ExpiresAt`), when the thing is large or shouldn't be copied (`*redis.Client`, `*pgxpool.Pool`), or when a method changes its receiver. Small structs like `Link` that are just handed back are fine as values.

**Q: What's wrong with `fmt.Errorf("...: %w", err)` on its own line?**
Ans: `fmt.Errorf` builds and **returns** a new error. It doesn't log it or change `err`. Calling it without using the result throws the wrapped error away, and the original unwrapped error is what gets returned. The fix is `return Link{}, fmt.Errorf("JSON marshal error: %w", err)`.

Follow-up: *Why wrap with `%w` at all?*
Ans: It adds context about where the error happened, while still letting callers check the original with `errors.Is` / `errors.As`.

**Q: How do you write a 24-hour duration in Go?**
Ans: `24 * time.Hour`. `time.Hour` is a `Duration` constant, and multiplying it by a number gives another `Duration`.

**My question (while wiring `CachedStore` into `main.go`):** *How does `cachedStore.LinkStore = &PostgresStore{db: pool}` work? Isn't `LinkStore` a different type from `*PostgresStore`?*
Why I asked: I was filling in `CachedStore`'s fields and was assigning a `*PostgresStore` to a field declared as `LinkStore`. They looked like two unrelated types, so I didn't expect it to compile.

Ans: `LinkStore` doesn't say what a store **is**. It says what a store can **do**: any type with both of those methods (`CreateLink`, `GetLink`), with exactly those signatures, counts as a `LinkStore`.

In `repository.go` there are `func (pgStore *PostgresStore) CreateLink(...)` and `func (pgStore *PostgresStore) GetLink(...)`. `*PostgresStore` has both methods, so it satisfies `LinkStore`, and the compiler lets you assign it to a field of that type.

Follow-up: *How is this different from C#?*
Ans: In C# you declare it yourself: `class PostgresStore : ILinkStore`. Go has no `implements` keyword. Satisfaction is **implicit**: the compiler checks the method set when you assign. `PostgresStore` never mentions `LinkStore`.

Follow-up: *What does the interface field actually hold?*
Ans: Two things: the dynamic type (`*PostgresStore`) and the value (the pointer). Calling `c.LinkStore.GetLink(...)` looks at the stored type and calls `(*PostgresStore).GetLink`. This is dynamic dispatch.

Follow-up: *Why is the `&` required?*
Ans: The methods have pointer receivers, so only `*PostgresStore` has them in its method set. A plain `PostgresStore` value doesn't satisfy `LinkStore`, and without the `&` the build fails with "method ... has pointer receiver". The same rule means `Server.db` needs `&cachedStore`.

Follow-up: *How do you catch a broken implementation early?*
Ans: Add a compile-time assertion next to the type: `var _ LinkStore = (*PostgresStore)(nil)`. It costs nothing at runtime and fails the build as soon as the type stops satisfying the interface.

---

## 6. Short code generation at scale (random vs counter vs hash)

**Q: How does Snip generate codes today?**
Ans: 7 random base62 characters from `crypto/rand`, inserted straight away. The `UNIQUE` constraint on `short_code` catches a collision (Postgres error `23505`), and we retry with a new code, up to `maxCodeAttempts = 5`. The database is the single source of truth for "is this taken?". There's no check-then-insert race, because the insert *is* the check.

**Q: How big is the keyspace?**
Ans: 62⁷ = 3,521,614,606,208, about **3.5 trillion** codes.

**Q: What's the chance a new random code collides?**
Ans: Roughly `existing links ÷ keyspace`:

| Links already in DB | Chance one insert collides | Roughly |
|---|---|---|
| 1 million | 2.8 × 10⁻⁷ | 1 in 3.5 million |
| 100 million | 2.8 × 10⁻⁵ | 1 in 35,000 |
| 1 billion | 2.8 × 10⁻⁴ | 1 in 3,500 |

Follow-up: *Is 5 retries enough?*
Ans: Yes, by a huge margin. All 5 attempts fail with probability p⁵. At 1 billion links that's (2.8 × 10⁻⁴)⁵ ≈ 10⁻¹⁸, which effectively never happens. Retries only become a problem when the keyspace is a big fraction full. At 50% full, p⁵ ≈ 3% of creates would fail. Long before that you'd move to 8 characters (62⁸ ≈ 218 trillion).

Follow-up: *So collisions basically never happen?*
Ans: Wrong. A *single* insert rarely collides, but *some* collision somewhere is near-certain early on. That's the birthday paradox: there's a 50% chance of at least one collision once you have about √(2 · K · ln 2) ≈ **2.2 million** codes. That's exactly why the unique constraint and the retry loop exist. A random scheme without them is a bug waiting to happen.

**Q: What is the counter-based approach?**
Ans: Take a number that only ever goes up (a Postgres sequence) and convert it to base62. Different numbers always give different codes, so collisions are impossible by construction.

| Number | Code (our alphabet: a–z, A–Z, 0–9) |
|---|---|
| 0 | `a` |
| 1 | `b` |
| 61 | `9` |
| 62 | `ba` |
| 125 | `cb` |

**Q: How does base62 encoding work?**
Ans: The same way you'd convert decimal to binary by hand: repeated `% 62` gives the next digit and `/ 62` gives what's left. The digits come out least-significant first, so reverse them at the end. Example for 125: `125 % 62 = 1` → `b`, `125 / 62 = 2`. Then `2 % 62 = 2` → `c`, `2 / 62 = 0`, stop. That gives `bc`, reversed to `cb`. Edge case: 0 must return `a`, not an empty string, because the loop never runs for 0.

```go
func EncodeBase62(n uint64) string {
    if n == 0 {
        return string(alphabet[0])
    }
    var buf []byte
    for n > 0 {
        buf = append(buf, alphabet[n%62])
        n /= 62
    }
    slices.Reverse(buf)
    return string(buf)
}
```

**Q: How would you wire a counter into our schema?**
Ans: `short_code` is `NOT NULL`, so the number is needed *before* the insert. We can't reuse `links.id`: it's `GENERATED ALWAYS AS IDENTITY`, so we don't get the value until after the insert, and we can't supply it without `OVERRIDING SYSTEM VALUE`. Use a separate sequence (`CREATE SEQUENCE short_code_seq`), call `SELECT nextval('short_code_seq')`, encode the result in Go, then `INSERT`.

Follow-up: *Sequences have gaps (`nextval` is never rolled back). Does that matter?*
Ans: No. We only need numbers that are unique, not consecutive. A skipped number is just a code nobody uses.

Follow-up: *Would you keep the unique constraint?*
Ans: Yes. It's a cheap safety net. Mixing schemes can collide too: once the counter passes 62⁶ (about 56.8 billion) it produces 7-character codes, which can clash with the random ones already in the table.

**Q: What's wrong with counter-based codes?**
Ans:
- **They're guessable.** Codes in order (`a`, `b`, `c`, ...) let anyone walk through every link, including "unlisted" ones, and reveal how many links you create, which is business information. You can hide the order by shuffling the number before encoding: multiply by a large number coprime to 62⁷ and take the result mod 62⁷, which is a one-to-one mapping, or use a library like Sqids. That's obfuscation, not security.
- **They need one source of numbers.** Every server has to get its number from one place, so that place becomes a bottleneck and a single point of failure.

**Q: How do big systems scale a counter?**
Ans:
- **Range allocation.** Each API instance reserves a block of numbers at once, for example 1,000 using a sequence with `INCREMENT BY 1000`, and hands them out from memory. One DB call per 1,000 links. If an instance crashes, the rest of its block is lost, which is fine because gaps don't matter.
- **Ticket servers** (Flickr's approach). Dedicated DB servers whose only job is to give out IDs, often two of them with odd/even numbers for availability.
- **Key Generation Service (KGS).** Pre-generate random unique codes into a table offline. Servers claim batches of them. There's no collision check when a link is created, and codes stay unguessable. The KGS has to be highly available, and keys claimed by a server that crashes are lost, which is acceptable.
- **Snowflake-style IDs** (timestamp + machine ID + sequence) need no coordination, but they're 64-bit, about 11 base62 characters, which is too long for a *short* link.

**Q: What about hashing the long URL?**
Ans: Take something like `sha256(longURL)` and keep the first 7 base62 characters.
- Pro: the same URL always gives the same code, so duplicates are free and no state is needed.
- Con: a truncated hash collides at the same rate as random codes (birthday paradox again), and handling it is clumsier. You can't just pick a new random code. You have to add a salt or counter and hash again.
- Con: the same URL giving the same code is often *unwanted*. Two users shortening the same URL need separate links for their own analytics and expiry dates (Week 10 multi-tenancy).

**Q: Which did we choose for Snip, and why?**
Ans: Random 7-character codes with a unique constraint and a retry. They need no coordination, so any number of stateless API instances can create links in parallel (Week 8). They're not guessable. And the math shows retries stay negligible into the billions of links. If we got close to tens of billions, we'd move to 8 characters, or to a KGS if create volume made the retry round-trips costly.

**Interview summary: "How do you generate unique short codes?"**
- Random base62 with a DB unique constraint and a retry. 7 characters gives 3.5 trillion codes, about a 1-in-3,500 collision chance per insert even at 1 billion links, and 5 retries is effectively never exhausted.
- Collisions *will* happen (50% chance of one by about 2.2M links), so the unique constraint is required, not optional.
- A counter plus base62 never collides, but it's guessable and needs one source of numbers. Scale it with range allocation or a ticket server.
- A KGS gives the benefits of both (no collision check at request time, unguessable codes), at the cost of running one more service.
- Hashing gives free duplicate handling, but it's rarely what you want per user and still has to handle collisions.

---






## 7. Cache verification and latency: expected results

> ⚠️ **These are expected results worked out from the code, not measurements from my machine.** Redis wasn't installed when this was written. Before quoting any number in an interview or on a resume, run it for real (about 10 minutes once Redis is up) and replace this section with the actual output. An interviewer who asks "how did you measure that?" will see through a borrowed number.

**Q: What does `redis-cli MONITOR` show for two redirects of the same code?**
Ans: First request (cache miss): a `get` that finds nothing, then the DB read, then a `set`:
```
"get" "link:abc1234"
"set" "link:abc1234" "{\"LongURL\":\"https://example.com\",\"ExpiresAt\":null}" "ex" "86400"
```
Second request (cache hit): only the `get`, with no `set` and no Postgres query:
```
"get" "link:abc1234"
```
That pair of lines is the proof the cache works. (go-redis sends command names in lowercase. On a fresh connection you may also see a `hello` / `client setinfo` handshake first. That's normal.)

Follow-up: *Why does the `set` say `ex 86400` for a normal link but `px <milliseconds>` for an expiring one?*
Ans: go-redis uses `EX` (seconds) when the duration is a whole number of seconds, like 24h, and `PX` (milliseconds) otherwise. `time.Until(*ExpiresAt)` is almost never a whole number of seconds, so expiring links get `px`.

**Q: What does `TTL link:<code>` show?**
Ans:
- **Normal link:** just under **86400**, counting down (for example `86397` a few seconds later).
- **Link expiring in 10 minutes:** about **600**, because of the `min(c.ttl, time.Until(*ExpiresAt))` rule. The cache entry dies at the same moment as the link.
- **`-2`:** the key doesn't exist (never cached, or already expired).
- **`-1`:** the key has no expiry. This should never happen with our code. If it does, something called `Set` with a 0 duration.

**Q: What happens when you request a code that doesn't exist, twice?**
Ans (from tracing the current `cache.go`, before the fix):
1. **First request:** `get` finds nothing → Postgres returns `ErrLinkNotFound` → `set "link:zzzzzzz" "not-found" "ex" "60"` → **404**. Correct.
2. **Second request:** `get` returns the string `"not-found"`. That's not a `redis.Nil` error, so the code takes the **hit** branch and calls `json.Unmarshal("not-found")`, which fails with `invalid character 'o' in literal null (expecting 'u')` (JSON reads the `n` as the start of `null`). That wrapped error matches no case in `writeError`, so the user gets **500 internal_server_error**.

Same code, two requests, two different status codes. The negative cache turns a correct 404 into a 500 for 60 seconds. Fix: check for the sentinel before `json.Unmarshal` and return `ErrLinkNotFound`.

**Q: How much faster is a cache hit than a miss?**
Ans (typical ranges for everything on one machine, Go on Windows talking to Redis in WSL2 and Postgres on localhost):

| Path | What happens | Typical server-side latency |
|---|---|---|
| Miss | Redis GET + Postgres indexed lookup + Redis SET | ~1–3 ms |
| Hit | Redis GET + JSON decode | ~0.3–1 ms |

So on a laptop you'd expect roughly **2–4× faster**, not 50×.

Follow-up: *Why is the gain so small locally?*
Ans: Postgres is already fast for a single row found through a unique B-tree index, and that row's pages sit in `shared_buffers` (memory) after the first read. On localhost both stores cost about one network round-trip. The cache's real value shows up when:
- **Postgres is far away or busy.** Managed DBs in the cloud add network latency, and under load queries queue for pool connections. Then a miss can cost 5–20 ms while a hit stays at 1 ms or less.
- **You count throughput, not latency.** Every hit is a query Postgres never sees. With a 95% hit rate, the DB handles 1/20th of the redirect traffic, and that's what lets the system scale reads.

Follow-up: *How would you measure it properly?*
Ans: Load test the redirect endpoint (for example with `hey` or `k6`) with the cache on and then off, and compare **p50 and p99**, not just the average. Then look at the Postgres query rate during each run. The drop in DB queries is the headline number.

**Interview summary: "How did you prove the cache works, and what did it buy you?"**
- MONITOR shows one `get`+`set` on the first redirect and a single `get` after that, so the DB isn't touched on a hit.
- `TTL` shows 24h for normal links and the time left for expiring links, so the cache can never outlive the link.
- On a single machine, hits are a few times faster. In production the bigger win is offloading the DB: with a 95% hit rate, Postgres gets about 5% of redirect reads.
- Tracing the not-found path found a real bug: the negative-cache sentinel reached `json.Unmarshal` and turned 404s into 500s.

---

## Open exercises

- On create, how should the create path deal with the cache so a freshly generated code can never be stuck behind a cached "not found" entry?
- Is a cache hit's `ExpiresAt` ever stale? Explain why or why not, given how the TTL is set.
