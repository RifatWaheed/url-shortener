# Design TinyURL — my answer, based on Snip

Written as if I'm answering live in an interview. Everything here is grounded in what I actually built (Go + Postgres + Redis).

**How to pace it in a 45-minute interview:**

| Minutes | Section |
|---|---|
| 0–5 | 1. Requirements (ask questions, agree on scope) |
| 5–10 | 2. Capacity estimates |
| 10–15 | 3. API + 4. Schema |
| 15–25 | 5. Code generation |
| 25–35 | 6. Caching |
| 35–45 | 7. Scaling reads + bottlenecks, then questions |

---

## 1. Requirements

**Functional (what it does)**
- Given a long URL, return a short URL (`POST /shorten`).
- Given a short code, redirect to the long URL (`GET /{code}`).
- Links can optionally expire. An expired link stops redirecting.

**Out of scope for now** (say this out loud so the interviewer knows you thought about it):
- Custom aliases (`/my-brand`)
- Click analytics (that's a follow-up design; Snip adds it in Week 6)
- User accounts / per-user links (Week 10)

**Non-functional (how well it does it)**
- **Read-heavy.** A link is created once and clicked many times. I assume 100 reads per write. This one fact drives most of the design: optimise the redirect path above everything else.
- **Low latency on redirects.** Users are waiting on a blank page while we look up the code.
- **High availability.** If redirects are down, every link anyone has ever shared is broken.
- **Codes should not be guessable.** Otherwise anyone can walk through all links, including ones meant to be private.
- **Durable.** Once we return a short URL, it must keep working until it expires.

---

## 2. Capacity estimates

**Assumptions:** 100M new links per month, 100:1 read:write ratio, links kept for 5 years.

| What | Calculation | Result |
|---|---|---|
| Writes/sec (average) | 100M ÷ 2.6M seconds in a month | **~40/s** |
| Writes/sec (peak, ×3) | | ~120/s |
| Reads/sec (average) | 40 × 100 | **~4,000/s** |
| Reads/sec (peak, ×3) | | ~12,000/s |
| Links after 5 years | 100M × 12 × 5 | **~6 billion** |
| Bytes per row | `id` 8 + `short_code` ~8 + `long_url` ~100 + `user_id` 8 + 2 timestamps 16 + row header ~24 ≈ 170 B. Round up to **500 B** to cover the indexes and bloat | 500 B |
| Storage after 5 years | 6B × 500 B | **~3 TB** |
| Reads per day | 4,000 × 86,400 | ~350M |
| Cache size (80/20 rule: cache the hottest 20% of daily reads) | 350M × 20% × ~200 B per Redis entry | **~14 GB** (upper bound: many of those reads repeat the same codes) |
| Bandwidth out | 4,000/s × ~500 B per redirect response | ~2 MB/s, which is negligible |

**What the numbers tell us:**
- 40 writes/sec is easy for a single Postgres primary. **Writes are not the problem.**
- 4,000–12,000 reads/sec on one Postgres is doable but leaves no headroom. **Reads are the problem, so cache them.**
- With a 95% cache hit rate, Postgres sees about 200 reads/sec + 40 writes/sec. That's comfortable.
- 14 GB of cache fits on one Redis node.
- 3 TB fits on one Postgres server. Sharding isn't needed for storage alone within 5 years.
- 6B links out of a 3.5 trillion keyspace is only about 0.2% full. 7-character codes are enough (see section 5).

---

## 3. API

### `POST /shorten`
Request:
```json
{ "url": "https://example.com/some/very/long/path" }
```
Response `201 Created`:
```json
{ "shortCode": "aZ3k9Qx", "shortUrl": "https://sho.rt/aZ3k9Qx" }
```

| Status | When |
|---|---|
| 201 | Created |
| 400 | URL missing, too long (> 2048 chars), unparseable, not http/https, or no host; invalid JSON |
| 413 | Body larger than 1 MB |
| 415 | Content-Type isn't `application/json` |
| 503 | Couldn't generate a unique code after 5 attempts (should practically never happen) |

### `GET /{code}`

| Status | When |
|---|---|
| 302 Found | Redirect to the long URL (`Location` header) |
| 404 | Code doesn't exist, or is malformed (wrong length or characters; a malformed code can never exist, so it's reported the same way) |
| 410 Gone | Link existed but has expired |

All errors share one JSON shape: `{"error": {"code": "link_not_found", "message": "..."}}`.

**Why 302 and not 301?**
A 301 (permanent) is cached by browsers. After the first click, the browser goes straight to the long URL and never asks us again. That's less load, but we lose the ability to count clicks (analytics), to expire links, or to change where a link points. A 302 (temporary) means every click comes through us. For a product that sells analytics, 302 is the right call. If pure redirect speed were the only goal, 301 would win.

---

## 4. Schema

```sql
CREATE TABLE links (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    short_code  VARCHAR(16) NOT NULL UNIQUE,
    long_url    TEXT NOT NULL,
    user_id     BIGINT REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ
);
```

**Design choices:**
- **`UNIQUE` on `short_code`** does two jobs. It creates the B-tree index that every redirect uses (`WHERE short_code = $1` is an index lookup, not a table scan), and it's what catches code collisions, because the insert itself is the uniqueness check.
- **A separate `id` primary key** as well as `short_code`. Other tables (like `clicks`) reference links by a compact 8-byte integer instead of a string, and the internal ID stays separate from the public code.
- **`expires_at` can be NULL.** NULL means "never expires". In Go it's a `*time.Time`, and `nil` maps to NULL.
- **`long_url` is `TEXT`,** and the length limit is enforced in the app (2048 chars). The limit is a business rule, so it lives in validation where it gives a clear error message.
- **`user_id` can be NULL** for now (anonymous links). It's ready for multi-tenancy in Week 10.

**Why Postgres and not a NoSQL key-value store?**
The access pattern is basically key-value, so DynamoDB or Cassandra would also work and would shard more easily. But at about 3 TB and 40 writes/sec, one Postgres handles it fine, and we get transactions, constraints, and joins for the analytics and users tables later. I'd switch only if the write volume or data size outgrew a single primary.

---

## 5. Code generation

**What Snip does:** 7 random base62 characters (`a–z A–Z 0–9`) from `crypto/rand`, inserted directly. If the `UNIQUE` constraint fires (Postgres error `23505`), it generates a new code and retries, up to 5 times.

**Why it's safe:**
- Keyspace: 62⁷ ≈ **3.5 trillion** codes.
- Collision chance for one insert = existing links ÷ keyspace. Even at 1 billion links, that's about **1 in 3,500**.
- All 5 retries failing at 1 billion links is about 10⁻¹⁸. Effectively never.
- But collisions *will* happen somewhere. The birthday paradox gives a 50% chance of at least one by about 2.2 million links. So the unique constraint plus retry is **required**, not optional.

**The alternatives I considered:**

| Approach | How | Pros | Cons |
|---|---|---|---|
| **Random** (chosen) | Random 7 chars + retry | No coordination between servers, not guessable | Collision odds grow with the table, so there's a retry loop |
| **Counter + base62** | Postgres sequence → encode the number | Never collides | Guessable (`a`, `b`, `c`…), and one central counter is a bottleneck |
| **Hash of URL** | First 7 chars of `sha256(url)` | Same URL gives the same code (free dedupe) | Still collides (truncated hash), awkward to fix; dedupe is unwanted when users need separate links |
| **Key Generation Service** | Pre-generate random unique codes offline; servers claim batches | No collision check at request time, not guessable | One more service to run and keep highly available |

**Why random:** any number of stateless API servers can create links in parallel with no coordination, codes aren't guessable, and the math shows retries stay negligible into billions of links.

**What would change at bigger scale:** near tens of billions of links, move to 8 characters (62⁸ ≈ 218 trillion). If create volume got so high that the retry round-trips mattered, move to a Key Generation Service. If we ever needed a counter, scale it with **range allocation**: each server reserves a block of 1,000 numbers at once instead of hitting the DB for every link.

---

## 6. Caching

**Strategy: cache-aside (lazy loading) with Redis.**
On a redirect: check Redis for `link:<code>`. On a hit, return it. On a miss, read Postgres, write the result into Redis with a TTL, return it.
- Why not write-through (cache every link at creation)? Most links are clicked a few times then forgotten. Write-through slows every create and fills the cache with cold links. Lazy loading only caches what's actually being clicked.

**Implementation:** a `CachedStore` that wraps the Postgres store and satisfies the same `LinkStore` interface (the decorator pattern). Handlers don't know the cache exists. Turning caching on is a one-line change in `main.go`.

**TTL rule:** `min(24h, time until the link expires)`. The cache entry dies at the same moment as the link, so the cache can never serve an expired link. Edge case: if the link expires between the DB read and the cache write, serve it but don't cache it, because a zero TTL in go-redis means "never expire".

**Negative caching (cache penetration):** requests for codes that don't exist would always miss the cache and hit Postgres, which is an easy way to hammer the DB. So "not found" is cached too, as a marker value with a short TTL (60s).
- Only a genuine "not found" is cached, never a DB error. Otherwise a brief outage would turn into a minute of false 404s.
- The read path checks for the marker *before* JSON-decoding. A bug here turned correct 404s into 500s.
- It doesn't stop an attacker using fresh random codes. That needs rate limiting (Week 6) and, at huge scale, a Bloom filter.

**Redis down?** Fail open: log a warning and go straight to Postgres. The cache is an optimisation, not a dependency for correctness. The risk is that all the load lands on Postgres at once, which is a reason to keep Postgres sized for some miss traffic.

**Cache stampede (thundering herd):** a hot link's key expires and 1,000 requests miss at once, all hitting Postgres for the same row. Fixes: `singleflight` (one DB query per key per server), a short Redis lock, or random jitter on TTLs so keys don't expire together.

**Effect on load (from section 2):** with a 95% hit rate, Postgres handles about 200 of the 4,000 reads/sec.

---

## 7. Scaling reads

> Week 8 covers this properly. For now, the two parts I can defend from my own code:

**1. Stateless API, so run more copies.** The Go server keeps nothing in memory between requests: links live in Postgres, the cache in Redis. Any instance can serve any request, so scaling out is just running more instances behind a load balancer.

**2. Redis absorbs most reads.** See section 6: about 95% of redirects never reach Postgres.

**To be expanded after Week 8:**
- **Read replicas:** read-only copies of Postgres for cache misses, so the primary only handles writes.
- **CDN / edge caching:** answer hot redirects from servers close to the user.
- **Sharding by `short_code`:** split the table across databases. Only needed if writes or storage outgrow one primary, which section 2 says won't happen within 5 years at this scale.

---

## Bottlenecks and known gaps (good to raise yourself before the interviewer does)

- **Single Postgres primary** is a single point of failure for writes. Fix: a replica with automatic failover.
- **Single Redis node.** If it dies, all reads fall to Postgres. Fix: Redis replica / Sentinel, and keep Postgres able to handle a burst.
- **Expired links aren't negatively cached,** so every click on an expired link still queries Postgres. Fix: cache a "gone" marker the same way as "not found".
- **A new code could be stuck behind a "not found" marker** if someone requested that exact code in the 60s before it was created. Very unlikely with random 7-char codes. The fix is an open exercise in `week4-notes.md`.
- **No rate limiting yet** (Week 6), so one client can flood `POST /shorten` or probe random codes.
