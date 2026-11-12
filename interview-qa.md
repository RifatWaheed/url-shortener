# Snip — Interview Q&A Bank

Everything I built and learned in the Snip project, as interview questions with model answers, grouped by topic.

**How to use this with a friend:**
- The friend reads the **bold question** out loud. I answer without looking. The friend compares my answer with the **Ans**.
- *Follow-up* lines are what a real interviewer asks next. Use them to push me.
- **📍 Code** points at the file where I built it. A strong answer says "in my project, I did X in Y".
- A ⭐ marks questions that come up in almost every backend interview. Drill those first.
- The final sections hold the numbers to memorise, the bugs I found (behavioral material) and the open questions I still need to answer.

This file grows as the project grows. Each finished week adds its topics.

---

## Contents

1. [Go language fundamentals](#1-go-language-fundamentals)
2. [HTTP API design](#2-http-api-design)
3. [Middleware, logging and panic recovery](#3-middleware-logging-and-panic-recovery)
4. [Config, startup and graceful shutdown](#4-config-startup-and-graceful-shutdown)
5. [Testing in Go](#5-testing-in-go)
6. [Postgres internals](#6-postgres-internals)
7. [Schema design and data types](#7-schema-design-and-data-types)
8. [Indexes and EXPLAIN](#8-indexes-and-explain)
9. [Joins and the query planner](#9-joins-and-the-query-planner)
10. [Transactions, isolation levels and deadlocks](#10-transactions-isolation-levels-and-deadlocks)
11. [Migrations, repository pattern, SQL injection](#11-migrations-repository-pattern-sql-injection)
12. [Caching with Redis](#12-caching-with-redis)
13. [Short code generation](#13-short-code-generation)
14. [System design: Design TinyURL](#14-system-design-design-tinyurl)
15. [Docker: images, layers, builds](#15-docker-images-layers-builds)
16. [Docker networking and ports](#16-docker-networking-and-ports)
17. [Docker Compose: multi-service stacks](#17-docker-compose-multi-service-stacks)
18. [DSA concepts](#18-dsa-concepts)
19. [Numbers to remember](#19-numbers-to-remember)
20. [Bugs I found and fixed (behavioral / "hardest bug" material)](#20-bugs-i-found-and-fixed)
21. [Open questions to answer before the interview](#21-open-questions-to-answer-before-the-interview)
22. [Coming next (filled in as the project continues)](#22-coming-next)

---

## 1. Go language fundamentals

**⭐ Q: How do interfaces work in Go, and how is that different from C#?**
Ans: An interface describes what a type can **do**: a set of method signatures. Any type that has those methods satisfies it **implicitly**. There's no `implements` keyword. `*PostgresStore` has `CreateLink` and `GetLink`, so it's a `LinkStore` without ever mentioning `LinkStore`. In C# you declare `class PostgresStore : ILinkStore`. In Go, the compiler checks the method set when you assign.
📍 Code: `LinkStore` in `handlers.go`, implemented by `PostgresStore` (`repository.go`), `CachedStore` (`cache.go`) and `fakeStore` (`handlers_test.go`).

Follow-up: *What does an interface value hold at runtime?*
Ans: Two things: the dynamic type (e.g. `*PostgresStore`) and the value (the pointer). A method call looks up the stored type and calls its method. That's dynamic dispatch.

Follow-up: *How do you catch a type that stops satisfying an interface?*
Ans: A compile-time assertion: `var _ LinkStore = (*fakeStore)(nil)`. It costs nothing at runtime and breaks the build immediately. I use it in `handlers_test.go`.

Follow-up: *Why did `cachedStore.LinkStore = PostgresStore{...}` (no `&`) fail?*
Ans: The methods have **pointer receivers**, so only `*PostgresStore` has them in its method set. A plain value doesn't satisfy the interface.

**Q: What is struct embedding, and how did you use it?**
Ans: Putting a type inside a struct without a field name. Its methods get **promoted** to the outer struct. `CachedStore` embeds `LinkStore`, so `CreateLink` goes straight to the real store, and I only override `GetLink`. That's the **decorator pattern**: same interface, added behavior, and the handlers don't know the cache exists.
📍 Code: `cache.go`, `type CachedStore struct { LinkStore; ... }`.

**⭐ Q: Explain pointers. When do you use a pointer vs a value?**
Ans: A pointer holds an **address**. `&x` = "address of x", `*p` = "the value at p". Use a pointer when:
- a function must change the caller's variable (`json.Unmarshal(data, &link)`, `Scan(&link.LongURL)`). Go passes everything by value, so without `&` the function fills a copy.
- "no value" must be possible: `ExpiresAt *time.Time`, where `nil` = never expires = SQL `NULL`.
- the thing is large or must not be copied (`*pgxpool.Pool`, `*redis.Client`).
- a method changes its receiver.

Small structs that are just returned (like `Link`) are fine as values.

Follow-up: *What happens if you dereference a nil pointer?*
Ans: A runtime panic (`nil pointer dereference`). That's why `GetLink` checks `link.ExpiresAt != nil` before `*link.ExpiresAt`.

Follow-up: *What does `**Link` mean, and how did you hit it?*
Ans: A pointer to a pointer. I declared `var link *Link`, then passed `&link`. Each `&` adds a layer. The fix was `var link Link` and `&link`.

**⭐ Q: How does error handling work in Go? What's `%w`?**
Ans: Errors are ordinary values returned as the last result, and the caller checks `if err != nil`. `fmt.Errorf("ping database: %w", err)` wraps an error: it adds context but keeps the original inside, so callers can still check it with `errors.Is` (match a sentinel like `ErrLinkNotFound`) or `errors.As` (get a typed error like `*pgconn.PgError`).
📍 Code: `writeError` in `response.go` maps sentinel errors to HTTP statuses with `errors.Is`. `CreateLink` uses `errors.As` to read the Postgres error code `23505`.

Follow-up: *What's wrong with calling `fmt.Errorf("...: %w", err)` on its own line?*
Ans: It **returns** a new error and doesn't log anything. If you ignore the result, the wrapped error is thrown away. I made this mistake once (see §20).

Follow-up: *Sentinel errors vs error types?*
Ans: Sentinels (`var ErrLinkNotFound = errors.New(...)`) are for "which error is it?" checks. Error types (`*pgconn.PgError`, `*http.MaxBytesError`) carry extra data (code, limit). I use both.

**⭐ Q: Goroutines and channels in your own words. Channels vs mutex?**
Ans: A goroutine is a function running concurrently, very cheap (a few KB of stack), scheduled by the Go runtime onto OS threads. A channel is a typed pipe between goroutines: send blocks until there's room (or a receiver), receive blocks until there's a value. A buffered channel `make(chan error, 1)` holds values without a waiting receiver.
- **Channels** for **passing ownership of data / signalling** between goroutines ("here's the result", "stop now"). Example: the server goroutine sends its error on `errCh`.
- **Mutex** for **protecting shared state** that several goroutines read/write in place (a counter, a map).
Rule of thumb: "share memory by communicating" when data flows; a mutex when it's just guarding a variable.
📍 Code: `main.go`, `errCh := make(chan error, 1)` + `select`.

**Q: What does `select` do?**
Ans: Waits on several channel operations and runs whichever is ready first. In `main.go` it waits for either the server failing (`errCh`) or a shutdown signal (`ctx.Done()`).

**⭐ Q: What is `context.Context` for?**
Ans: Carrying **cancellation, deadlines and request-scoped values** through a call chain. I pass `r.Context()` from the handler into the store, so if the client disconnects, the DB/Redis call is cancelled instead of wasting work. `context.WithTimeout` puts a deadline on the startup ping (5 s) and on graceful shutdown (10 s). `signal.NotifyContext` gives a context that's cancelled on Ctrl+C / SIGTERM.

**Q: What are `defer`, `panic` and `recover`?**
Ans: `defer` schedules a call to run when the function returns (in reverse order). `panic` unwinds the stack. `recover`, called inside a deferred function, stops the unwinding and returns the panic value. My `recoverPanic` middleware uses this so one buggy request returns a 500 instead of crashing the whole server.

**Q: You used generics. Where and why?**
Ans: `DecodeJSONBody[T any](w, r) (T, error)` decodes any request type with the same checks (content type, size cap, unknown fields, single object). Without generics I'd write it once per request type or use `any` and type-assert.
📍 Code: `validations.go`.

**Q: `crypto/rand` vs `math/rand` — which did you use for short codes, and why?**
Ans: `crypto/rand`. Codes must be **unguessable**, otherwise someone can predict and enumerate links. `math/rand` is fast but predictable if you know the seed/state.

**Q: Your short code generator skips bytes ≥ 248. Why?**
Ans: **Modulo bias.** A random byte is 0–255 (256 values). 256 isn't a multiple of 62, so `b % 62` would make the first 8 characters (`256 - 248 = 8`) slightly more likely. 248 = 62 × 4, so keeping only bytes < 248 makes every character exactly equally likely. This is called rejection sampling.
📍 Code: `shortcode.go`.

**Q: Go modules — what are `go.mod` and `go.sum`?**
Ans: `go.mod` declares the module path, Go version and direct dependencies with versions. `go.sum` records cryptographic hashes of every dependency so builds are verifiable and reproducible. Commit both.

---

## 2. HTTP API design

**⭐ Q: Describe your API.**
Ans:
| Endpoint | Success | Errors |
|---|---|---|
| `POST /shorten` `{"url": "..."}` | `201` + `{"shortCode", "shortUrl"}` | `400` invalid JSON/URL, `413` body > 1 MB, `415` wrong Content-Type, `503` no unique code after 5 tries |
| `GET /{code}` | `302` + `Location` header | `404` not found or malformed, `410` expired |
| `GET /health` | `200` | — |

Every error has one shape: `{"error": {"code": "link_not_found", "message": "..."}}`. `code` is a stable machine-readable string for clients; `message` is for humans.
📍 Code: `server.go` (routes), `handlers.go`, `response.go`.

**⭐ Q: Why 302 and not 301 for the redirect?**
Ans: Browsers **cache 301** permanently. After the first click they go straight to the long URL and never ask us again. That reduces load, but we lose click analytics, link expiry, and the ability to change the target. With 302 every click comes through us. For a product that sells analytics, 302 is right.

**Q: Why return 404 (not 400) for a malformed short code?**
Ans: A malformed code can never exist, so from the client's view it's simply "not found". It also doesn't tell an attacker which validation rule they broke. The validation still runs first, so Postgres and Redis are never queried for garbage.

**Q: Why 410 Gone for an expired link?**
Ans: It tells the client the link *did* exist but is permanently gone, which is more accurate than 404. Crawlers drop 410s faster.

**⭐ Q: Walk me through how you validate `POST /shorten`.**
Ans: In layers, cheapest first:
1. `Content-Type` must be `application/json` (parsed with `mime.ParseMediaType`, so `; charset=utf-8` is OK) → else `415`.
2. `http.MaxBytesReader` caps the **body** at 1 MB → else `413`. This stops someone streaming gigabytes into the decoder.
3. `DisallowUnknownFields` rejects typos like `{"ulr": ...}` → `400`.
4. `decoder.More()` rejects two objects in one body → `400`.
5. Then the **field** rules: non-empty, ≤ 2048 characters, parses with `url.ParseRequestURI`, scheme `http`/`https`, has a host.
📍 Code: `DecodeJSONBody` and `ValidateUrl` in `validations.go`.

Follow-up: *Body cap vs field length cap — aren't they the same thing?*
Ans: No. The **body cap** (1 MB) protects the server: it limits how much is read off the network at all. The **field cap** (2048) is a business rule about what a valid URL is. A 1 MB body could still hold a 900 KB URL, which the field cap rejects with a clear message.

Follow-up: *Why only http/https?*
Ans: Otherwise someone can shorten `javascript:` or `file:` URLs and use our trusted domain to deliver them.

**Q: How does Go 1.22+ routing work here? Why doesn't `/health` get treated as a short code?**
Ans: Patterns can include a method and wildcards: `"GET /{code}"`, read with `r.PathValue("code")`. When two patterns match, the **most specific** wins. `/health` is more specific than `/{code}`, so it goes to the health handler.

**Q: Why are there timeouts on `http.Server`?**
Ans: Without them, a slow or malicious client can hold a connection open forever (slowloris), and enough of them exhaust the server. `ReadHeaderTimeout` (5 s) is the key slowloris guard. `ReadTimeout`, `WriteTimeout` and `IdleTimeout` bound the rest of the request lifecycle.
📍 Code: `main.go`.

**Q: Why must `Content-Type` be set before `WriteHeader`?**
Ans: Headers are sent with the status line. After `WriteHeader` they're already on the wire, so later changes are ignored. `writeJSON` sets the header, then the status, then encodes the body.

**⭐ Q: Offset vs keyset (cursor) pagination?** *(concept; `GET /links` isn't built in Snip yet)*
Ans: Offset (`LIMIT 20 OFFSET 10000`) is simple but slow on deep pages, because Postgres still reads and discards the first 10 000 rows, and rows shift when data changes between pages. Keyset (`WHERE id < $last_seen_id ORDER BY id DESC LIMIT 20`) uses the index to jump straight to the next page, so it stays fast at any depth and is stable under inserts. The cost is no "jump to page 57". Use keyset for feeds and large lists.

---

## 3. Middleware, logging and panic recovery

**Q: What is middleware in Go?**
Ans: A function that takes an `http.Handler` and returns a new `http.Handler` that does something before/after calling the next one. They chain: `logRequests(recoverPanic(mux))`.
📍 Code: `middleware.go`, wired in `server.go`.

**Q: Why is `logRequests` outside `recoverPanic`, not inside?**
Ans: A request runs through the outer layer first. If a handler panics, `recoverPanic` catches it and writes a 500 **through the logger's status recorder**, so the log line shows `status: 500`. If the order were reversed, the panic would unwind past the logger before it could log, and the request would never be logged.

**Q: How does the logger know the response status? `http.ResponseWriter` doesn't expose it.**
Ans: I wrap the writer in a `statusRecorder` that embeds `http.ResponseWriter` and overrides `WriteHeader` to remember the code. It also overrides `Write`, because a handler that calls `Write` without `WriteHeader` gets an implicit 200. It only records the **first** status, because HTTP only sends the first one.

Follow-up: *Why does `statusRecorder` have an `Unwrap()` method?*
Ans: Wrapping hides optional interfaces of the real writer (like `Flush` or `Hijack`). `http.ResponseController` calls `Unwrap()` to reach the original writer, so wrapping doesn't break streaming or WebSockets later.

**Q: In `recoverPanic`, why re-panic on `http.ErrAbortHandler`?**
Ans: `net/http` uses that specific panic on purpose to abort a response silently. Swallowing it would turn an intentional abort into a logged 500.

**⭐ Q: Why structured logging (`slog` with JSON) instead of `fmt.Println`?**
Ans: Each log line is a JSON object with fields (`method`, `path`, `status`, `duration_micro_sec`). Log tools (Loki, CloudWatch, Datadog) can filter and aggregate on fields, like "all 5xx on `/shorten` in the last hour" or "p99 duration". Free text needs regexes. I log at `Error` only for 5xx, so client mistakes (4xx) don't flood the error logs.

---

## 4. Config, startup and graceful shutdown

**⭐ Q: How does your app get its configuration?**
Ans: Environment variables (`DATABASE_URL`, `BASE_URL`, `ADDR`, `REDIS_ADDR`). Locally `godotenv` loads them from `.env`. In Docker they come from `environment:` in compose. That's the 12-factor rule: config belongs to the environment, not the code or the image, so the same binary/image runs on my laptop, in CI and in prod.

Follow-up: *What if a required variable is missing?*
Ans: The app **fails fast** at startup with a clear message (`DATABASE_URL is required`). Optional ones (`ADDR`, `REDIS_ADDR`) have safe defaults via `getEnv`. Failing at boot is much better than failing on the first request in production.

Follow-up: *Why is `BASE_URL` separate from `ADDR`?*
Ans: `ADDR` is where the server **listens** (`0.0.0.0:8080` in a container). `BASE_URL` is what **users** see in short links (`https://sho.rt`). Behind a proxy, container or load balancer they're different.

**Q: Why ping the database at startup?**
Ans: `pgxpool.New` doesn't actually connect. Pinging (with a 5 s timeout) proves the DB is reachable and the credentials work, so a broken config fails at deploy time, not on the first user request.

**⭐ Q: What is graceful shutdown and how did you implement it?**
Ans: On SIGTERM (what Docker, Kubernetes and most hosts send before killing a process), stop accepting new connections, let in-flight requests finish, then exit. Otherwise every deploy cuts off requests mid-flight.
1. `signal.NotifyContext` gives a context cancelled on SIGINT/SIGTERM.
2. The server runs in a goroutine; `select` waits for a server error or the signal.
3. `httpSrv.Shutdown(ctx)` with a 10 s deadline drains requests.
4. `defer pool.Close()` and `defer rdb.Close()` close connections on the way out.
📍 Code: `run()` in `main.go`.

Follow-up: *Why is `errCh` buffered with size 1?*
Ans: When we shut down via the signal path, nobody reads `errCh` any more. `ListenAndServe` then returns `ErrServerClosed` and its goroutine sends on the channel. With an unbuffered channel that send would block forever, which is a **goroutine leak**. Size 1 lets it send and exit.

Follow-up: *Why `run() error` instead of putting everything in `main()`?*
Ans: `os.Exit` skips deferred calls. With `run()`, all the `defer`s (close pool, close Redis) run first, and `main` does the single `os.Exit(1)` at the end. It also makes the startup logic testable.

---

## 5. Testing in Go

**⭐ Q: Unit vs integration tests?**
Ans: A **unit test** checks one piece in isolation with dependencies faked, so it's fast and deterministic (my handler tests with `fakeStore`). An **integration test** runs against real dependencies (a real Postgres) and proves the SQL, constraints and wiring actually work. You want many unit tests and fewer, more expensive integration tests (the test pyramid).

**⭐ Q: What's a table-driven test?**
Ans: A slice/map of cases (`name`, input, expected output) and one loop that runs each with `t.Run(name, ...)`. Adding a case is one line, every case gets its own name in the output, and you can run one with `go test -run TestHandlers/expired`.
📍 Code: `handlers_test.go`, `validations_test.go`.

**⭐ Q: How do you test HTTP handlers without starting a server or a database?**
Ans: `httptest.NewRequest` builds a request, `httptest.NewRecorder` captures the response, and I call the router's `ServeHTTP` directly. The DB is replaced by a `fakeStore` that implements `LinkStore` and returns whatever the test case wants (a code, `ErrLinkNotFound`, `ErrLinkExpired`, a raw "connection refused" error). That's only possible because the handlers depend on the **interface**, not on `*pgxpool.Pool`.

Follow-up: *What do the handler tests cover?*
Ans: Success paths (201 with the right body, 302 with the right `Location`), every error mapping (404, 410, 413, 415, 400 variants, 503), and that an unknown DB error becomes a generic 500 without leaking internals.

**Q: How did you test the middleware?**
Ans: `TestRecoverPanic` passes a handler that panics and asserts a 500 JSON response. `TestStatusRecorderKeepsFirstStatus` checks that a second `WriteHeader` doesn't overwrite the first status.

**Q: How do you test something random like `GenerateShortCode`?**
Ans: Test **properties**, not exact values. `TestGenerateShortCode` generates 10,000 codes and checks that each has length 7, passes `ValidateShortCode` (only alphabet characters), and none repeats (tracked in a `map[string]struct{}`). At 10,000 codes in a 3.5 trillion keyspace, a collision in the test is astronomically unlikely, so a failure would point to a real bug like a broken random source.

---

## 6. Postgres internals

**Q: How does Postgres handle client connections?**
Ans: Process per connection. The postmaster forks a **backend process** for each client, and it parses, plans and executes that client's SQL. Processes are expensive (memory, fork time), which is why apps use a **connection pool** (`pgxpool` in Snip, PgBouncer in front of the DB at scale) instead of opening a connection per request.

**Q: What's in shared memory?**
Ans: **Shared buffers**, a cache of 8 KB data pages in RAM. Every read and write goes through it first. Plus **WAL buffers**, which hold WAL records waiting to be written.

**⭐ Q: What is the WAL and why does it exist?**
Ans: Write-Ahead Log: an append-only, sequential log in `pg_wal`. Every change is recorded in the WAL **before** the data page changes. On `COMMIT`, only the WAL has to be flushed to disk (`fsync`); the changed data pages can stay in memory. If Postgres crashes, it replays the WAL from the last checkpoint. It gives **durability** (D in ACID) with good **performance**, because sequential writes are far cheaper than random page writes. The same WAL is streamed to replicas (replication) and archived for point-in-time recovery.

**Q: What does the checkpointer do?**
Ans: Gradually flushes dirty pages from shared buffers to the data files (spread out to avoid I/O spikes), then writes a checkpoint record to the WAL. Crash recovery only needs to replay from the last checkpoint, and older WAL can be recycled. It runs every `checkpoint_timeout` (5 min default) or when the WAL grows past `max_wal_size`.

**Q: How are tables stored on disk?**
Ans: As **heap files** of 8 KB pages, split into 1 GB segments. A page has a header, line pointers, free space, and tuples (rows) stored from the end. Rows are in **no particular order** (no clustering by primary key). Each tuple has a `ctid` (block, index) = its physical location, plus `xmin`/`xmax` transaction IDs. Indexes are separate files mapping key → `ctid`.

**Q: What's TOAST?**
Ans: How Postgres stores large values, since a row can't span pages. Above ~2 KB it first **compresses** variable-length values (`TEXT`, `JSONB`, `BYTEA`), and if that's not enough, moves them **out of line** into a TOAST table in ~2 KB chunks, leaving a pointer in the row. Fixed-size types (`BIGINT`, `TIMESTAMPTZ`) are never toasted. Max value size is 1 GB.

**⭐ Q: Explain MVCC.**
Ans: Multi-Version Concurrency Control. Postgres never overwrites a row in place. An `UPDATE` marks the old version dead (sets its `xmax` to the updating transaction's ID) and inserts a new version (`xmin` = that transaction ID). A `DELETE` only sets `xmax`. Each transaction reads from a **snapshot** and sees only the versions visible to it based on `xmin`/`xmax`. The result: **readers never block writers and writers never block readers.** Writers do block other writers on the **same row**.

Follow-up: *What's the downside?*
Ans: Dead tuples. Every update leaves an old version behind (table bloat), which **VACUUM** has to clean up.

Follow-up: *What's a HOT update?*
Ans: Heap-Only Tuple: if no indexed column changes and the page has room, the new version goes on the same page and **indexes aren't touched**. That's one reason not to index columns you update often.

**⭐ Q: What does VACUUM do? VACUUM vs VACUUM FULL?**
Ans: VACUUM marks space from dead tuples (no longer visible to any transaction) as reusable, updates the visibility map and free space map, and **freezes** old tuples to prevent transaction ID wraparound (IDs are 32-bit). It doesn't return space to the OS and doesn't lock the table. **VACUUM FULL** rewrites the whole table and returns space to the OS, but takes an **exclusive lock**, so the table is unusable meanwhile. Autovacuum runs VACUUM automatically. `ANALYZE` (often run with it) refreshes the planner's statistics.

---

## 7. Schema design and data types

**⭐ Q: Walk me through your schema.**
Ans: Four tables: `users`, `links`, `clicks`, `email_verifications`.
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
`clicks` references `links(id)` and stores `clicked_at`, `referrer`, `user_agent`, `ip_address INET`, `country`, with an index on `link_id`.
📍 Code: `migrations/`.

**Q: `BIGINT GENERATED ALWAYS AS IDENTITY` vs `SERIAL`?**
Ans: Identity is the SQL-standard way. `SERIAL` is a Postgres shortcut that creates a separate sequence with awkward ownership/permission behavior. `GENERATED ALWAYS` also stops anyone inserting their own ID by accident (you'd need `OVERRIDING SYSTEM VALUE`). `BIGINT` because `INT` runs out at ~2.1 billion rows, and changing a primary key's type later on a big table is painful.

**Q: Why a separate `id` when `short_code` is already unique?**
Ans: Other tables (`clicks`, potentially millions of rows) reference links by a compact 8-byte integer instead of a string. The internal ID also stays separate from the public code, so the code could change without breaking references.

**Q: Why `TIMESTAMPTZ` and not `TIMESTAMP`?**
Ans: `TIMESTAMPTZ` stores an absolute instant (in UTC) and converts to the session's time zone on display. `TIMESTAMP` stores a wall-clock time with no zone, so "10:00" from servers in different zones becomes ambiguous. Use `TIMESTAMPTZ` for events.

**Q: `TEXT` vs `VARCHAR(n)` in Postgres?**
Ans: Same storage and performance. `VARCHAR(n)` just adds a length check. I use `TEXT` for `long_url` and enforce 2048 in the app, because that's a business rule with a clear error message. `VARCHAR(16)` on `short_code` is a sanity cap.

**Q: Why is `expires_at` nullable?**
Ans: `NULL` means "never expires". In Go it's `*time.Time` and `nil` maps to `NULL`. A magic date like 9999-12-31 would be a hack that every query has to know about.

**Q: What does `ON DELETE CASCADE` do on `links.user_id`?**
Ans: Deleting a user automatically deletes their links (and through `clicks.link_id`, their clicks). The alternatives are `RESTRICT` (block the delete) or `SET NULL` (keep the rows orphaned). Cascade fits "delete my account means delete my data".

**⭐ Q: Postgres auto-indexes primary keys and unique columns. Does it index foreign keys?**
Ans: **No.** That's why I added `idx_clicks_link_id` by hand. Without it, "all clicks for link X" scans the whole `clicks` table, and deleting a link (cascade) must also scan `clicks` to find its children.

**Q: Why `INET` for the IP address?**
Ans: It validates the value, stores IPv4/IPv6 compactly, and supports network operators (e.g. "is this IP in 10.0.0.0/8?"). Text would accept garbage.

**Q: Why store a `token_hash` in `email_verifications`, not the token?**
Ans: Same reason as passwords: if the database leaks, raw tokens would let an attacker verify accounts. Store a hash, compare hashes. `used_at` makes tokens single-use, and `expires_at` limits the window.

---

## 8. Indexes and EXPLAIN

**⭐ Q: What is a B-tree index and how does it speed up a lookup?**
Ans: A balanced tree of sorted keys pointing to row locations (`ctid`). A lookup walks from the root to a leaf in a few page reads (O(log n)) instead of scanning every page of the table. It supports `=`, `<`, `>`, `BETWEEN`, `ORDER BY` and prefix `LIKE 'abc%'` (with the right operator class). It's the default index type.
📍 Code: the `UNIQUE` on `short_code` creates the B-tree that every redirect uses.

Follow-up: *Other index types?*
Ans: **Hash** (equality only), **GIN** (arrays, JSONB, full-text: "which rows contain this element"), **GiST** (geometry, ranges, nearest-neighbor), **BRIN** (tiny summaries per block range, great for huge append-only tables ordered by time, like `clicks.clicked_at`).

**⭐ Q: When does an index NOT help?**
Ans:
- **Functions on the column:** `WHERE lower(short_code) = 'sc42'` can't use a plain index on `short_code`, because the index stores the raw values. Fix: an expression index on `lower(short_code)`.
- **Leading wildcard:** `LIKE '%42'` has no prefix to jump to, so it's a seq scan. (A trigram GIN index can help.)
- **Low selectivity:** `WHERE is_verified = true` when 95% of rows are true. Jumping around via the index to read almost every row is slower than one sequential pass, so the planner picks a seq scan.
- **Tiny tables:** a few pages are cheaper to scan than to traverse an index.

**⭐ Q: When does an index HURT?**
Ans: On writes. Every `INSERT` (and every `UPDATE` of an indexed column) must update **every** index, not just the table. Five extra indexes on `links` make every `POST /shorten` slower and use more disk and memory. Indexes also block HOT updates. Index what you query, nothing more.

**⭐ Q: How do you read an `EXPLAIN ANALYZE` plan?**
Ans:
```
Index Scan using links_short_code_key on links (cost=0.29..8.31 rows=1 width=72)
                                              (actual time=0.015..0.016 rows=1 loops=1)
  Index Cond: (short_code = 'abc123'::text)
  Buffers: shared hit=3 read=1
```
- **Node type:** Seq Scan, Index Scan, Index Only Scan, Bitmap Heap Scan, Nested Loop, Hash Join, Merge Join, Sort, Aggregate, Limit.
- **cost=startup..total:** in arbitrary units (1.0 = one sequential page read), not milliseconds.
- **rows:** estimated rows the node **outputs** (after filtering), not rows scanned.
- **actual time / rows / loops:** real milliseconds per loop; multiply by `loops`.
- **Buffers:** `shared hit` = found in shared buffers; `read` = came from the OS cache or disk.

Read it **inside-out, bottom-up** (the most-indented node runs first). Look for the node with the biggest actual time, a 10× gap between estimated and actual rows (bad statistics → `ANALYZE`), seq scans on big tables, a large "Rows Removed by Filter", and `read` much higher than `hit`.

**Q: What surprised you in your own EXPLAIN experiments?**
Ans: Two things.
1. A 1-row table used the index before `ANALYZE`. With no statistics (`reltuples = -1`), the planner assumes the table is at least 10 pages, so a seq scan looked more expensive than the index.
2. Stale stats aren't frozen stats. At plan time the planner takes the table's **current** page count and multiplies it by the old rows-per-page density, so estimates still grow with the table. What actually goes stale is the density and the data distribution.

**Q: When does autovacuum run ANALYZE?**
Ans: When changed rows exceed `autovacuum_analyze_threshold + autovacuum_analyze_scale_factor × table rows` (defaults: 50 + 10%). On huge tables 10% is a lot of change, so you lower the scale factor per table.

**Q: What's an index-only scan?**
Ans: When every column the query needs is in the index, Postgres can answer from the index alone without visiting the heap, but only for pages the **visibility map** marks all-visible. That's another reason VACUUM matters.

---

## 9. Joins and the query planner

**⭐ Q: Inner vs left join vs anti join?**
Ans:
- **INNER JOIN:** only rows with a match on both sides. "Links that have at least one click, with their clicks."
- **LEFT JOIN:** every row from the left, with matches or NULLs. "Every link with its click count, including 0."
- **ANTI JOIN:** left rows with **no** match. "Links never clicked." Written as `WHERE NOT EXISTS (SELECT 1 FROM clicks c WHERE c.link_id = l.id)` or `LEFT JOIN ... WHERE c.id IS NULL`.

Follow-up: *Why avoid `NOT IN (subquery)` for anti joins?*
Ans: If the subquery returns a single `NULL`, `NOT IN` returns no rows at all (three-valued logic). `NOT EXISTS` doesn't have that trap and the planner turns it into a real anti join.

Follow-up: *Common LEFT JOIN mistake?*
Ans: Filtering the right table in `WHERE` (`WHERE c.clicked_at > ...`) silently turns it into an inner join, because NULL rows fail the condition. Put that filter in the `ON` clause.

**⭐ Q: What join algorithms does Postgres use, and when?**
Ans:
- **Nested loop:** for each outer row, look up matches in the inner side. Great when the outer side is small and the inner side has an index (e.g. 10 links → index lookup on `clicks.link_id`). Terrible for two big tables without an index.
- **Hash join:** build a hash table from the smaller side, then stream the larger side through it. The go-to for big, unsorted equality joins. Needs memory (`work_mem`); spills to disk if it doesn't fit.
- **Merge join:** both sides sorted on the join key, walk them together. Good when both inputs are already sorted (e.g. from indexes) or very large.

Follow-up: *Why does the planner pick one over another?*
Ans: It estimates the cost of each option from **statistics** (row counts, value distributions, correlation) and picks the cheapest. Bad estimates (stale stats) are the usual reason it picks a bad plan, for example a nested loop over what turns out to be millions of rows.

---

## 10. Transactions, isolation levels and deadlocks

**⭐ Q: What is ACID?**
Ans: **Atomicity:** all of a transaction's changes happen, or none. **Consistency:** constraints hold before and after. **Isolation:** concurrent transactions don't see each other's partial work (to a degree set by the isolation level). **Durability:** once committed, it survives a crash (the WAL is flushed on commit).

**⭐ Q: Explain the read anomalies.**
Ans (bank account with balance 100):
- **Dirty read:** reading another transaction's **uncommitted** change. A sets 50 then rolls back, but B already used 50.
- **Non-repeatable read:** reading the same row twice in one transaction and getting different values, because someone committed in between (100, then 50).
- **Phantom read:** running the same query twice and getting a different **set** of rows (`balance > 80` returns 3, then 4 after someone inserts).
- **Serialization anomaly:** each transaction is valid alone, but the combined result couldn't happen in any one-at-a-time order. Example: the rule is "at least one doctor on call". Both doctors check, each sees the other on call, and both go off call.

**⭐ Q: The four isolation levels — and what Postgres actually does?**
Ans:
| Level | What you see | In Postgres |
|---|---|---|
| Read Uncommitted | Others' uncommitted changes (dirty reads) | Behaves like Read Committed; Postgres never allows dirty reads |
| **Read Committed** (default) | A new snapshot for **each statement** | Non-repeatable reads and phantoms possible |
| Repeatable Read | One snapshot for the **whole transaction** | Also prevents phantoms in Postgres. Updating a row someone else changed → serialization error, abort |
| Serializable | As if transactions ran one at a time | Detects dangerous patterns (like the doctors) and aborts one transaction |

Follow-up: *What must the app do at Repeatable Read / Serializable?*
Ans: **Retry** on serialization failures (SQLSTATE `40001`). Those levels trade blocking for aborting, so the app needs a retry loop.

**⭐ Q: What is a deadlock and how do you avoid it?**
Ans: Two transactions each hold a lock the other needs: T1 locks row A and wants B, T2 locks B and wants A. Neither can proceed. Postgres detects the cycle (after `deadlock_timeout`, 1 s) and aborts one with an error.
Avoid it by:
- **Locking in a consistent order** (e.g. always by ascending ID). Then a cycle can't form.
- **Keeping transactions short.** No network calls or user waits inside a transaction.
- **Locking what you need up front** (`SELECT ... FOR UPDATE` in a known order).
- **Retrying** the aborted transaction, since deadlocks can still happen.

**Q: Pessimistic vs optimistic locking?**
Ans: **Pessimistic:** lock first (`SELECT ... FOR UPDATE`), then change. Safe under heavy contention, but holds locks. **Optimistic:** read a `version` column, then `UPDATE ... WHERE id = $1 AND version = $2`. If 0 rows were updated, someone else changed it, so retry. No locks held, best when conflicts are rare.

**Q: Where does Snip rely on the database for correctness?**
Ans: Short-code uniqueness. I don't "check if the code exists, then insert". Two requests could both pass the check and then both insert (a race). Instead I just `INSERT`. The `UNIQUE` constraint is the check, enforced atomically by Postgres. On error `23505` (unique violation) I generate a new code and retry.
📍 Code: `CreateLink` in `repository.go`.

---

## 11. Migrations, repository pattern, SQL injection

**⭐ Q: Why use a migration tool instead of running SQL by hand?**
Ans: The schema becomes versioned code in git. Every environment (my laptop, CI, prod) applies the same numbered steps in order, and the tool records which ones ran. I use `golang-migrate` with `up`/`down` pairs in `migrations/`.

Follow-up: *What's the `schema_migrations` table?*
Ans: The tool's logbook: the current version and a `dirty` flag. Rerunning applies only new migrations (`no change` otherwise), so migrations are **idempotent** and safe on every deploy. If one fails halfway, `dirty = true` blocks further runs until a human checks the database.

Follow-up: *What are down migrations for, and do you trust them in prod?*
Ans: They undo a step, which is handy in development. In prod, rolling back often means losing data (dropping a column), so teams usually **roll forward** with a new fixing migration instead.

**Q: What is the repository pattern and why use it?**
Ans: All SQL lives behind an interface (`LinkStore`) implemented by `PostgresStore`. Handlers call `CreateLink`/`GetLink` and know nothing about SQL or pgx. This lets me swap in a cache decorator (`CachedStore`) or a fake for tests without touching handlers.

**⭐ Q: Why is your app safe from SQL injection?**
Ans: Every query is **parameterized**: `WHERE short_code = $1` with the value sent separately. The SQL text and the data travel separately to Postgres, so user input is never parsed as SQL, whatever it contains. String concatenation (`"... WHERE short_code = '" + code + "'"`) is what creates injection. Input validation on top is defense in depth, not the main protection.

---

## 12. Caching with Redis

**⭐ Q: What caching strategy did you use, and why?**
Ans: **Cache-aside** (lazy loading). On a redirect: check Redis for `link:<code>`. Hit → return it. Miss → read Postgres, write to Redis with a TTL, return. Only links that are actually clicked get cached.
📍 Code: `CachedStore.GetLink` in `cache.go`.

Follow-up: *Why not write-through?*
Ans: Write-through caches every link at creation. That slows every write and fills the cache with links that may never be clicked. A shortener is read-heavy (1 create, many redirects), so lazy loading fits.

Follow-up: *Other strategies?*
Ans: **Write-behind** (write to cache, flush to the DB later: fast but risks data loss) and **read-through** (the cache library itself loads from the DB on a miss: same idea as cache-aside, but inside the cache layer).

**⭐ Q: How do you choose the TTL?**
Ans: `min(24h, time until the link expires)`. The cache entry dies at the same moment as the link, so the cache can never serve an expired link. If the link has already expired between the DB read and the cache write, I serve it but **don't cache** it, because in go-redis a TTL of 0 means **no expiry**, which would pin a dead link forever.

**⭐ Q: What happens if Redis goes down?**
Ans: **Fail open.** The `Get` error (anything other than `redis.Nil`) is logged as a warning and the request falls through to Postgres. A failed `Set` is only logged. Redirects keep working, just slower. The cache is an optimization, not a correctness dependency.

Follow-up: *Risk?*
Ans: All the traffic the cache was absorbing lands on Postgres at once. Postgres must be sized for some miss traffic, or protected with rate limiting / a circuit breaker.

**⭐ Q: How do you handle cache invalidation?**
Ans: Links are basically immutable once created, so there's little to invalidate, and the TTL bounds staleness. If links became editable or deletable, I'd **delete the key** on update (not update it), so the next read reloads fresh data. Deleting avoids a race where two writers set the cache in the wrong order.

**⭐ Q: What is cache penetration and how do you defend against it?**
Ans: Requests for keys that don't exist anywhere. They always miss, so every one hits the DB. An attacker can use made-up codes to hammer Postgres. Layers, cheapest first:
1. **Input validation:** malformed codes never reach Redis or Postgres.
2. **Rate limiting per client** (Week 6).
3. **Negative caching:** cache "not found" as a marker with a **short TTL** (60 s), so repeats of the same bad code skip the DB.
4. **Bloom filter:** answers "definitely doesn't exist" for any code, at huge scale.

Follow-up: *Why a short TTL for negative entries?*
Ans: The code might be created soon. A long negative TTL would make a new link 404.

Follow-up: *Which errors do you negatively cache?*
Ans: Only a genuine `ErrLinkNotFound`. Never infrastructure errors: if Postgres blips and I cache "not found", real links 404 for the whole TTL, turning a 1-second blip into a 60-second outage.

Follow-up: *Does negative caching stop random-code attacks?*
Ans: No. Each fresh random code is a first-time miss, and now also a junk key in Redis. That needs rate limiting and, at scale, a Bloom filter.

**Q: What's a Bloom filter? Why didn't you add one?**
Ans: A bit array plus several hash functions. If any of a key's bits is 0, the key is **definitely absent**. If all are 1, it's **maybe present**. False positives, never false negatives. I didn't add one because: every create must update it (if that fails, real links become false 404s), standard Bloom filters can't delete (expired links pile up), and it needs the RedisBloom module. It's worth it when attacks are **distributed** (bypassing per-IP limits) and the keyspace is huge.

**⭐ Q: What is a cache stampede (thundering herd)? How do you prevent it?**
Ans: A hot key expires and hundreds of requests miss at the same moment, all querying Postgres for the same row. Fixes:
- **Request coalescing:** `singleflight` in Go, so only one request per key per server goes to the DB and the others wait for its result.
- **Distributed lock:** `SET lock:<code> NX PX <ms>`, so one instance in the whole fleet rebuilds the key.
- **TTL jitter:** add randomness so keys cached together don't expire together.
- **Early refresh:** refresh hot keys shortly before they expire.

**Q: Why prefix keys with `link:`?**
Ans: Namespacing. The same Redis will hold rate-limit counters and more. Prefixes prevent collisions and let you scan/delete one kind of key.

**Q: How would you prove the cache works?**
Ans: `redis-cli MONITOR` shows `get` + `set` on the first redirect and only `get` on the second (no DB call). `TTL link:<code>` shows ~86400 for normal links and the remaining lifetime for expiring ones (`-2` = key gone, `-1` = no expiry, which should never happen in my code).

**Q: How much faster is a cache hit?**
Ans: On one machine, only about 2–4×, because Postgres is already fast for a single indexed row in shared buffers. The real win is **throughput**: with a 95% hit rate, Postgres handles 1/20th of redirect reads. That's what lets reads scale. To measure properly, load-test with the cache on and off (`hey`, `k6`) and compare **p50/p99** latency and the DB query rate.

---

## 13. Short code generation

**⭐ Q: How does Snip generate short codes?**
Ans: 7 random base62 characters (`a–z A–Z 0–9`) from `crypto/rand`, inserted directly. The `UNIQUE` constraint catches a collision (error `23505`) and I retry with a new code, up to 5 times (`ErrCodeExhausted` → 503).
📍 Code: `shortcode.go`, `CreateLink` in `repository.go`.

**⭐ Q: How big is the keyspace and how likely are collisions?**
Ans: 62⁷ ≈ **3.5 trillion**. The chance one insert collides ≈ existing links ÷ keyspace: 1 in 3.5 million at 1M links, about **1 in 3,500 at 1 billion**. All 5 retries failing at 1B links is ~10⁻¹⁸, effectively never.

Follow-up: *So collisions basically never happen?*
Ans: Wrong. By the **birthday paradox** there's a 50% chance of at least one collision somewhere by ~**2.2 million** codes. That's why the unique constraint + retry is required, not optional.

**⭐ Q: Compare random vs counter vs hash vs KGS.**
Ans:
| Approach | How | Pros | Cons |
|---|---|---|---|
| **Random** (mine) | Random 7 chars + retry | No coordination between servers, unguessable | Retry loop; odds grow with table size |
| **Counter + base62** | Sequence number → base62 | Never collides | Guessable/enumerable, leaks volume, one central counter |
| **Hash of URL** | First 7 chars of `sha256(url)` | Same URL → same code (free dedupe) | Truncated hash still collides; dedupe is unwanted per user |
| **Key Generation Service** | Pre-generate unique random codes, servers claim batches | No collision check at request time, unguessable | Another service to run and keep highly available |

**Q: How does base62 encoding of a number work?**
Ans: Repeatedly take `n % 62` (next digit) and `n / 62` (the rest), then reverse, because digits come out least-significant first. Edge case: 0 must return the first character, not an empty string.

**Q: How do you scale a counter if you need one?**
Ans: **Range allocation:** each server reserves a block (e.g. 1,000 numbers) and hands them out from memory, so one DB call per 1,000 links. Gaps are fine. Or **ticket servers** (Flickr: dedicated ID DBs, odd/even pairs). **Snowflake IDs** need no coordination but are 64-bit (~11 base62 chars), too long for a short link.

**Q: Why did you choose random?**
Ans: Any number of stateless API instances can create links in parallel with no coordination, codes aren't guessable, and the math shows retries stay negligible into billions of links. Near tens of billions I'd go to 8 characters (62⁸ ≈ 218 trillion), or a KGS if retry round-trips started to matter.

---

## 14. System design: Design TinyURL

*Full write-up: `week4-design-tinyurl.md`. This is the condensed version to say out loud in about 5 minutes, then go deep where the interviewer points.*

**⭐ Q: Design a URL shortener.**
Ans:
1. **Requirements.** Shorten (`POST /shorten`) and redirect (`GET /{code}`), optional expiry. Non-functional: **read-heavy** (~100:1), low-latency redirects, highly available, unguessable codes, durable. Out of scope at first: custom aliases, analytics, accounts. Say these out loud.
2. **Estimates.** 100M links/month → ~40 writes/s, ~4,000 reads/s (peak ~12k). 5 years → ~6B links × ~500 B ≈ **3 TB**. Cache ~20% of daily hot reads ≈ **~14 GB**, which fits on one Redis node. Conclusion: **writes are easy, reads are the problem.**
3. **API + schema.** See §2 and §7. 302 not 301 (analytics, expiry). `UNIQUE(short_code)` is both the redirect index and the collision check.
4. **Code generation.** Random 7-char base62 + unique constraint + retry (§13).
5. **Caching.** Cache-aside in Redis, TTL = min(24h, time to expiry), negative caching for misses, fail open if Redis dies, stampede protection (§12). With a 95% hit rate, Postgres sees ~200 reads/s.
6. **Scaling reads.** The API is **stateless** (all state in Postgres/Redis), so add instances behind a load balancer. Add Postgres **read replicas** for cache misses. A CDN/edge for the hottest links. **Sharding** by `short_code` only if writes or storage outgrow one primary, which won't happen within 5 years at this scale.
7. **Bottlenecks I'd raise myself.** Single Postgres primary (→ replica + failover), single Redis (→ replica/Sentinel), expired links aren't negatively cached, no rate limiting yet (→ Week 6).

Follow-up: *Why Postgres and not a NoSQL key-value store?*
Ans: The access pattern is key-value, so DynamoDB/Cassandra would work and shard more easily. But at ~3 TB and 40 writes/s one Postgres is fine, and I get transactions, constraints and joins for users and analytics. I'd switch if write volume or size outgrew one primary.

Follow-up: *How do you pace 45 minutes?*
Ans: Requirements 0–5, estimates 5–10, API + schema 10–15, code generation 15–25, caching 25–35, scaling + bottlenecks 35–45.

---

## 15. Docker: images, layers, builds

**⭐ Q: Image vs container?**
Ans: An **image** is a read-only template: a stack of filesystem layers plus config (entrypoint, user, env, ports). A **container** is a running (or stopped) instance: the image plus a thin **writable layer** and a process. One image can start many containers. Changing a file inside a container never changes the image.

**Q: What is a layer, and why are layers shared?**
Ans: The filesystem change from one instruction that touches files (`FROM`, `COPY`, `RUN`). Each layer is identified by a **hash of its contents**, so identical layers are stored once and shared across images. `USER`, `EXPOSE`, `ENTRYPOINT` only change config and add no layer.

**Q: What is a tag? Does `latest` mean newest?**
Ans: A tag is a **mutable pointer** to an image, like a git branch name. `latest` is just the default label when you don't specify one. Nothing keeps it current. CI tags images with the **git commit SHA** so you always know exactly what runs in prod and can roll back by redeploying an older tag.

**Q: Why does every rebuild give a new image ID even with no changes?**
Ans: The ID shown is the digest of the **manifest list**, which includes a BuildKit **provenance attestation** with build-time details. That changes every build even when every layer is identical. To compare contents, compare `RootFS.Layers`, not IDs.

**⭐ Q: How does the build cache work, and how did you order your Dockerfile for it?**
Ans: Steps are checked top to bottom. Once one step misses, **every step after it re-runs**. So: stable things first, changing things last. I copy `go.mod`/`go.sum` and run `go mod download` **before** `COPY . .`, so editing Go code doesn't re-download dependencies. `.dockerignore` keeps notes, `.git`, `.env` and binaries out of the build context, so they can't break the cache or leak secrets.

**⭐ Q: What is a multi-stage build? Why do small images matter?**
Ans: Stage 1 (`golang:1.27-alpine`) has the toolchain and compiles the binary. Stage 2 (`alpine:3.24`) copies **only** the binary. Only the last stage ships. Mine went from **804 MB → 41 MB** (13 MB compressed). Small images mean faster pushes/pulls/deploys and scale-out, less disk, and a **smaller attack surface** (no compiler, no source, fewer packages with CVEs).

**Q: Why `CGO_ENABLED=0`?**
Ans: A fully static binary with no C library dependency, so it runs on minimal bases (alpine, distroless, even `scratch`). All my dependencies are pure Go.

**⭐ Q: Exec form vs shell form `ENTRYPOINT`?**
Ans: `ENTRYPOINT ["/bin/snip"]` (exec form) runs my binary as **PID 1**, so `docker stop`'s SIGTERM reaches it and graceful shutdown runs. Shell form (`ENTRYPOINT /bin/snip`) runs `/bin/sh -c`, the shell is PID 1 and doesn't forward the signal, so Docker waits 10 s and SIGKILLs it, cutting off in-flight requests.

**Q: Why run as a non-root user?**
Ans: Containers run as root by default. If the app is exploited, the attacker has root in the container, which makes escaping or tampering easier. `adduser` + `USER appuser` limits the damage.

**Q: Does `EXPOSE 8080` publish the port?**
Ans: No. It's documentation/metadata. Only `-p` / `ports:` publishes.

**Q: Why config via env vars and not a `.env` baked into the image?**
Ans: The same image must run everywhere with different settings, and baking in `.env` would put secrets into every copy of the image and every registry it's pushed to. I verified `.env` was not in the image (`.dockerignore`).

---

## 16. Docker networking and ports

**⭐ Q: Why can't a containerized app reach Postgres at `localhost`?**
Ans: Each container has its own **network namespace**, effectively its own machine. `localhost` inside it is the container itself. On Docker Desktop the host is `host.docker.internal`. In Compose, other containers are reached by **service name** (`postgres:5432`).

**⭐ Q: The app listened on `127.0.0.1:8080` and the port was published, but requests failed. Why?**
Ans: `127.0.0.1` accepts only connections from **inside the same machine**, here the container. Published traffic arrives from outside, on the container's network interface. Listening on `0.0.0.0:8080` (all interfaces) fixes it. I set it through `ADDR`, so `127.0.0.1` stays the safe default when running directly on my laptop.

**Q: How do Compose services find each other?**
Ans: Compose puts every service on a default network with **built-in DNS**: the service name is the hostname. The API uses `postgres:5432` and `redis:6379`. Containers talk to each other's **container ports** directly, never through published ports.
Analogy: containers are apartments, the network is the internal hallway, and `ports:` cuts a front door to the street.

**⭐ Q: Which services did you publish, and why only those?**
Ans: Only the API (`8080`). Postgres is published as `127.0.0.1:5433:5432`, loopback only, so just my PC (pgAdmin) can reach it, not the Wi-Fi. Redis isn't published at all. It has **no password by default**, and anyone who can write to it could `SET` a short code to an attacker's URL: **cache poisoning** that redirects real users. Rule: expose only what something outside actually needs.

**Q: What does the `127.0.0.1:` prefix in a port mapping do?**
Ans: Binds the published port to the host's loopback only. Without it Docker binds to `0.0.0.0` (every interface), and on Linux Docker's own iptables rules can even bypass `ufw`.

**Q: When would you split into several networks?**
Ans: For isolation. A public reverse proxy on a `frontend` network, Postgres/Redis on a `backend` network, and only the API on both. A compromised proxy then has no route to the database. Containers only reach each other through networks they share.

---

## 17. Docker Compose: multi-service stacks

**⭐ Q: What does Compose give you over `docker run`?**
Ans: The whole stack (API, Postgres, Redis, migrations), with config, networking, volumes and startup order, in one file, started with one command (`docker compose up`). The file **is** the documentation of how the system runs.
📍 Code: `docker-compose.yml`.

**Q: `image:` vs `build:`?**
Ans: `image:` pulls a ready-made image. `build: .` builds from my Dockerfile. With both, Compose builds and tags the result with the `image:` name (`snip-api`).

**Q: Why pin `postgres:18-alpine` instead of `postgres`?**
Ans: No tag = `latest`, which moves. It could jump to Postgres 19, and a data folder from 18 won't start under 19 without an upgrade. Pinning the major version gets patches but no surprise breaking change. `migrate/migrate:v4.20.1` is pinned exactly for full reproducibility. Plain tags (not `-amd64`) are **manifest lists**, so Intel and ARM machines each get the right build.

**⭐ Q: How do migrations run in your Compose stack?**
Ans: A separate one-shot `migrate` service: it mounts `./migrations`, runs `up` against `postgres:5432`, and exits `0`. `Exited (0)` is success, not a crash.

Follow-up: *Why not put the SQL files in Postgres's `/docker-entrypoint-initdb.d`?*
Ans: initdb runs **every** `.sql` file alphabetically, including the `.down.sql` files, and it only runs on an empty data folder, so it can never apply a new migration to an existing database. A real migration tool tracks versions.

**⭐ Q: How do you keep database data across `docker compose down`?**
Ans: A **named volume**: `pgdata:/var/lib/postgresql`, declared under the top-level `volumes:`. The volume lives outside any container, so a new container mounts the same data. I proved it: created a link, `down`, `up`, and the row was still there.
Analogy: the container is the apartment, the volume is a storage unit outside the building.

Follow-up: *`down` vs `down -v`?*
Ans: `down` removes containers and the network but **keeps volumes**. `down -v` also deletes volumes: the deliberate "start from zero" button.

Follow-up: *Named volume vs bind mount?*
Ans: `./migrations:/migrations` (starts with `./`) is a **bind mount**: a folder on my PC that I edit. `pgdata:/...` (a plain name) is a **named volume** that Docker owns. Bind mounts for source/config I edit; named volumes for data the container owns.

Follow-up: *Why mount `/var/lib/postgresql` and not `/var/lib/postgresql/data`?*
Ans: From Postgres 18 the image stores data in a per-major-version subfolder. Mounting the parent lets a future 19 data folder sit next to 18 in the same volume (needed for `pg_upgrade --link`). The old path fails on PG 18.

Follow-up: *What are the hex-named volumes in `docker volume ls`?*
Ans: **Anonymous volumes**, created when an image declares a `VOLUME` (Redis declares `/data`) and nothing is mounted there. They pile up; `docker volume prune` removes unused ones.

**⭐ Q: `depends_on` was set, but the API still crashed with `connection refused` on startup. Why?**
Ans: Plain `depends_on` only waits for the dependency's container to **start**, not for the database inside to be **ready**. Postgres needed ~3 s to boot, and my app pings once and exits. `connection refused` (not a DNS error) showed the address was right but nothing was listening yet.

**⭐ Q: How did you fix it?**
Ans: **Healthchecks** + `depends_on` **conditions**.
- Postgres: `test: ["CMD", "pg_isready", "-h", "localhost", "-U", "postgres", "-d", "url_shortener"]`, Redis: `["CMD", "redis-cli", "ping"]`, with `interval`, `timeout`, `retries`, `start_period`.
- `migrate` waits for `postgres: service_healthy`.
- `api` waits for `postgres` and `redis` `service_healthy` and `migrate: service_completed_successfully`.

Result: the API waited ~7 s in `Created`, then connected on the **first try**, with zero errors.

Follow-up: *How does Docker decide "healthy"?*
Ans: Only the **exit code** of the test command: `0` = pass, anything else = fail. It ignores the output. `retries` failures in a row = `unhealthy`. `start_period` is a grace period at boot.

Follow-up: *Why `-h localhost` on `pg_isready`?*
Ans: On first start the Postgres image runs a **temporary server** that listens only on the Unix socket while it creates the user and database, then restarts. Without `-h`, `pg_isready` checks the socket and could report healthy during that phase. `-h localhost` forces TCP, which only the real server answers.

Follow-up: *How do you pick a healthcheck command?*
Ans: It must exist **inside** the image, check what dependents actually need ("accepts connections", not "process alive"), and be tried by hand first (`docker compose exec ... ; echo $LASTEXITCODE` → `0`).

**⭐ Q: The three `depends_on` conditions?**
Ans:
| Condition | Waits until | Use for |
|---|---|---|
| `service_started` (default) | Container started | Almost nothing; this caused the race |
| `service_healthy` | Healthcheck passes | Long-running servers (Postgres, Redis) |
| `service_completed_successfully` | Exited with code **0** | One-shot jobs (migrations) |

Follow-up: *How does Compose know `migrate` completed successfully?*
Ans: `migrate` just exits with a code, like any program (my own `main.go` does `os.Exit(1)` on error). Docker records it (`docker inspect ... {{.State.ExitCode}}`), and Compose reads it. Non-zero → Compose aborts and the **API never starts**. So migrations are a **gate**: the API can't run against a half-built schema.

**⭐ Q: Why did you remove `restart: on-failure`?**
Ans: It was hiding the startup race by crash-looping until Postgres happened to be ready. It can't tell "not ready yet" from "broken" (a wrong password would loop forever), and for `migrate` a dirty migration fails the same way on every retry. **Startup order belongs to healthchecks + `depends_on`. Restart policies are for runtime crashes** (e.g. `restart: unless-stopped` in prod, or the cloud host's equivalent).

**Q: Is it OK to have a password in `docker-compose.yml`?**
Ans: Here, yes: `snip_dev` is a throwaway credential for a DB that exists only inside Docker on my laptop. My real password was in `.env.docker`, which I removed from git (`.gitignore` + `git rm --cached`). In a real system, a secret that reached a remote is **leaked**: rotate it; deleting it from history isn't enough.

---

## 18. DSA concepts

**⭐ Q: Why is a hash map O(1) on average? When is it not?**
Ans: The key is hashed to a bucket index, so lookup goes straight to one bucket instead of searching. **Average** O(1) assumes a good hash function spreads keys evenly and the table **resizes** when the load factor gets high (Go grows the map, rehashing entries, which is O(n) occasionally but amortized O(1) per insert). **Worst case** O(n) when many keys land in the same bucket (bad hash or adversarial input). Go randomizes map iteration order and seeds its hash per map, partly to stop people depending on order and to resist hash-flooding attacks.

*(Problem patterns from the Friday DSA sessions: two pointers, sliding window, linked lists, stacks/queues, recursion/trees. Add the problems you solved and the pattern behind each one here as you go.)*

---

## 19. Numbers to remember

| Number | What |
|---|---|
| 62⁷ ≈ 3.5 trillion | Keyspace for 7-char base62 codes |
| ~2.2 million | Codes at which there's a 50% chance of at least one collision (birthday paradox) |
| 1 in 3,500 | Chance one insert collides at 1 billion links |
| 5 | Max code-generation attempts before 503 |
| 1 MB / 2048 chars | Request body cap / URL field cap |
| 24 h / 60 s | Positive cache TTL / negative cache TTL |
| 100:1 | Assumed read:write ratio |
| ~40/s writes, ~4,000/s reads | TinyURL estimates for 100M links/month |
| ~3 TB, ~14 GB | 5-year storage / cache size estimate |
| 95% hit rate → ~200 reads/s on Postgres | What the cache buys |
| 8 KB / 1 GB | Postgres page size / heap segment size and max field size |
| ~2 KB | TOAST threshold |
| 804 MB → 41 MB (13 MB compressed) | Docker image before/after multi-stage build |
| 5 s / 10 s / 10 s / 60 s | ReadHeader / Read / Write / Idle server timeouts |
| 10 s | Graceful shutdown deadline (also Docker's default stop grace before SIGKILL) |

---

## 20. Bugs I found and fixed

Each of these is a ready-made answer to "Tell me about a hard bug" or "a time you found a problem in production-like conditions". Use the STAR format: Situation, Task, Action, Result.

**1. The negative cache turned 404s into 500s.**
- **S/T:** I added negative caching: on "not found", store `"not-found"` in Redis for 60 s.
- **A:** By tracing two requests for the same missing code, I found that the second request got the string `"not-found"` from Redis. That's not a `redis.Nil` error, so the code took the **hit** path and called `json.Unmarshal("not-found")`, which failed. The wrapped error matched no case in `writeError`, so the user got a **500**.
- **R:** Same code, two requests, two different status codes. The fix: check for the sentinel **before** unmarshalling and return `ErrLinkNotFound`. Lesson: every new value you put in a shared store needs every reader updated.
📍 `cache.go`, `if val == linkNotFound`.

**2. The startup race hidden by a restart policy.**
- **S/T:** `docker compose up` "worked", but the API logs showed 4× `connection refused` first.
- **A:** I worked out that `depends_on` only waits for a container to *start*, and `restart: on-failure` was masking it by crash-looping. I added healthchecks + `depends_on` conditions and removed the restart policy so the test was honest. I also caught a subtle trap: `pg_isready` without `-h` can pass during Postgres's init-phase temporary server.
- **R:** Cold start with zero errors, and a failed migration now blocks the API instead of looping.

**3. Data lost on every `docker compose down`.**
- **S/T:** After `down`/`up`, the `links` table was empty.
- **A:** The data lived in the container's writable layer. I added a named volume at the PG 18 path.
- **R:** Verified that a link survives `down`/`up`; learned `down` vs `down -v`.

**4. A real password committed to git.**
- **S/T:** `.env.docker` held my real Windows Postgres password and was tracked.
- **A:** `.gitignore` + `git rm --cached`, moved the Docker stack to a throwaway dev password.
- **R:** Lesson: `.gitignore` doesn't affect already-tracked files, and in a real system a pushed secret must be **rotated**, because history rewrites don't un-leak it.

**5. A silently dropped error.**
- **S/T:** A `fmt.Errorf("...: %w", err)` call on its own line.
- **A/R:** `fmt.Errorf` returns a new error and does nothing else, so the context was thrown away. Fixed by returning it. Lesson: in Go, an error you don't return or log is gone.

**6. Caching an already-expired link forever.**
- **S/T:** TTL = time until expiry. If the link expired between the DB read and the cache write, the TTL would be ≤ 0.
- **A/R:** In go-redis, a TTL of 0 means **no expiry**, so an edge-case race would have become a permanent bug. Now I serve the link but skip caching when the remaining time is ≤ 0.

---

## 21. Open questions to answer before the interview

These came up during the build and were deliberately left open. Answer them in your own words before the mock interview.

1. *(Caching)* On create, how should the create path deal with the cache so a freshly generated code can never be stuck behind a cached "not found" entry?
2. *(Caching)* Is a cache hit's `ExpiresAt` ever stale? Explain why or why not, given how the TTL is set.
3. *(Compose)* `migrate` already waited for Postgres to be healthy and then finished. Why does `api` still wait for Postgres to be healthy too?
4. *(Docker)* How would you write `wget -qO- http://localhost:8080/health` as an exec-form healthcheck `test`, and why can't you use it on a `scratch` or distroless image?

---

## 22. Coming next

Sections to be added as the project continues (from the interview-readiness checklist):

- **Docker image size:** scratch vs distroless vs alpine, `-ldflags="-s -w"` (Week 5 Thu)
- **Click analytics + async writes:** event model, buffered channels, back-pressure, draining on shutdown (Week 6)
- **Rate limiting:** fixed window vs sliding window vs token bucket, distributed state in Redis; "Design a rate limiter" (Week 6)
- **CI/CD:** the pipeline step by step from commit to prod; CI vs CD; tagging by SHA; rollbacks; load balancing and statelessness (Week 7)
- **Queues:** why queues, Redis Streams and consumer groups, idempotency, retries, dead-letter queues; Go + Python pipeline (Week 9)
- **Auth + multi-tenancy:** JWT vs sessions, bcrypt, tenant scoping, OWASP basics, secrets management (Week 10)
- **Observability:** logs → metrics → traces, Prometheus, Grafana, request IDs; "how do you debug prod?" (Week 10)
- **Frontend + real-time:** Angular dashboard, SSE vs WebSockets (Week 11)
- **System design fundamentals:** read replicas, sharding, CDNs, CAP theorem with a real example (Week 12)
- **Behavioral:** STAR stories: a conflict, a failure, something you led, a hard bug (§20 has bug stories) (Week 12)
