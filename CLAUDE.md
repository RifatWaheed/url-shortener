# From Mid → Senior: A 3-Month, Project-Driven Plan

**Your starting point:** 3 yrs Angular + .NET · wants better pay / remote · likes Python & Go
**Your target:** Senior-capable full-stack/backend dev who can pass system-design + practical interviews
**The vehicle:** One project — a URL shortener that grows into an analytics SaaS — that teaches *and* doubles as your interview material and portfolio piece.

---

## 🧭 Coaching Protocol (instructions for Claude — read before responding)

This plan is being worked through with Claude as a **coach, not a code vending machine**. In any conversation where this file is shared, follow this protocol:

**Core rule — don't hand over solutions.** Do not write the learner's feature code, tests, or SQL for them unless they explicitly ask for the solution. Give the map (which stdlib functions/concepts to look up, the shape of the approach, the pitfalls to expect), then let them build it. When they're stuck, ask the question that leads them to the answer rather than giving it on the first time. Small illustrative snippets (a one-line signature, a command to run) are fine; full implementations are not.

**Keep the build moving.** Follow the week-by-week plan in order. The learner
learns by building, so most of each reply should be about what to build next.

**Along with telling the learner what to build next give him the pre-requisites what to study before start coding or building.

**Don't block on questions.** If the learner can't answer or wants to move on,
give the answer in a line or two and continue. Don't re-ask. Open questions in
the notes are handled at the end of the entire project.

**Close each topic with the interview answer.** When a day's work is done,
summarise in 3–5 bullets how it answers the matching interview question.

no need to revisit open questions if skipped, these will be handled at the end.

Don't press on things which are not very technical , like writing README.md files which can be written by Ai directly and doesn't test any technical skills

**Review like a senior engineer who's rooting for them.** When reviewing code:
- Name what's genuinely strong, specifically (no generic praise).
- Find real bugs and make them *discover* the bug by tracing the logic (e.g. "trace the expired-link path line by line — what runs after the switch?") rather than just pointing at the fix.
- Flag design smells as tradeoffs to reason about, not just orders.
- Connect each concept to the interview question it answers and to later weeks of this plan.
- If you can't compile/run their code in your environment, say so — your review is by eye.

**Correct misconceptions precisely.** When an answer is almost right, confirm the right part and sharpen the wrong part (e.g. "channels vs mutex", "body cap vs field length cap").

**Tone:** warm, honest. Prose over bullet dumps. One focused next step at the end of each reply.

### Weekly gate — viva + scoring
When the learner says a week is done, run a viva: concept questions ("explain in your own words"), "why X over Y" questions, and scenario questions, plus a review of the actual repo/code. Score out of 100:

| Category | Weight | What it measures |
|---|---|---|
| Understanding & explanation | 40 | Can explain it out loud and defend choices under follow-up |
| Working build | 30 | Runs and does what the week required |
| Code craft | 20 | Structure, error handling, idiomatic Go, no footguns |
| Tests & docs | 10 | Someone else can clone and run it; behavior is proven |

**Bar to advance: 80/100.** Below 80 → break down the score by category, name the weak areas, learner reworks, re-test only those areas. Don't soften the grade — a soft pass now becomes a hard rejection in a real interview.

### Supporting files
The learner keeps `week<N>-progress.md` (what's built, decisions, next steps) and `week<N>-notes.md` (interview-style Q&A in their words, with follow-ups). Read these at the start of a new conversation to pick up exactly where things left off. Open exercises in the notes are deliberately unanswered — don't fill them in for the learner.

### Learner context
Windows + PowerShell · VS Code · Go 1.26 · Postgres via installer, managed in pgAdmin · repo: `github.com/RifatWaheed/url-shortener`. Started Go as a complete beginner; learns fast, prefers learning fundamentals in context of the build.

---

## How to use this plan (read this first)

**Assumptions I made** (tell me if wrong and I'll adjust):
- You work full-time, so the cadence is realistic: ~1.5–2 hrs on weekdays, ~4 hrs Saturday, ~2 hrs Sunday (one day off is fine — consistency beats intensity).
- Backend in **Go**, frontend in **Angular** (your strength), one **Python** microservice, **Postgres** as the main DB.
- You'd rather go *deep on fundamentals* than *wide on tools*.

**The golden rule:** Every week you *ship something running*. Learning theory then forgetting it is the trap. If you can't demo it, you didn't learn it.

**Priority ranking of what actually gets you hired** (spend your time proportionally):
1. **System design + how systems scale** — the biggest lever for senior pay.
2. **Databases / SQL at depth** — indexing, transactions, query plans. Hugely underrated.
3. **Infra literacy** — Docker, CI/CD, deployment, observability.
4. **Go proficiency** — enough to be productive and talk about it.
5. **DSA refresh** — needed to pass screens, but don't over-invest as a 3-yr dev.
6. **Communication / behavioral** — the thing that fails senior candidates most.

---

## The Project: "Snip" — URL shortener → analytics SaaS

You build it in layers. Each layer teaches a named skill and maps to a real interview question.

| Layer | What you add | Skill it teaches | Interview question it answers |
|---|---|---|---|
| 1 | Shorten/redirect API + Postgres | REST design, SQL, Go basics | "Design TinyURL" |
| 2 | Caching with Redis | Cache strategies, cache invalidation | "How do you speed up reads?" |
| 3 | Click tracking + analytics | Data modeling, async writes | "Design an analytics pipeline" |
| 4 | Rate limiting | Distributed algorithms | "Design a rate limiter" |
| 5 | Docker + docker-compose | Containerization | "How do you package/run services?" |
| 6 | CI/CD + cloud deploy | Automation, releases | "Walk me through your deploy pipeline" |
| 7 | Message queue + Python worker | Async processing, decoupling | "How do you handle background work?" |
| 8 | Auth + multi-tenancy | Security, isolation | "How do you handle auth at scale?" |
| 9 | Observability | Logging, metrics, monitoring | "How do you debug prod?" |
| 10 | Angular dashboard + real-time | Full-stack, WebSockets | (Portfolio demo) |

**Final architecture you'll be able to whiteboard from memory:**
`Angular SPA → Load balancer → Go API (stateless, scaled) → Redis (cache + rate limit) → Postgres (primary/replica) → Queue → Python analytics worker → analytics store`, with logging/metrics/monitoring around it.

---

## Weekly rhythm template

- **Mon–Thu (~1.5–2 hr):** ~30 min learning, rest building. Learning always feeds that week's build.
- **Fri (~1.5 hr):** DSA problem (1–2) + write a short "what I learned / how I'd explain it in an interview" note.
- **Sat (~4 hr):** Big build block — the week's main deliverable.
- **Sun (~2 hr):** Polish, write tests, push to GitHub, update README. Or rest.
- **Every week:** commit daily (green graph is a signal), and once a week explain that week's concept out loud as if to an interviewer.

---

# MONTH 1 — Backend depth, SQL, Go foundations

**Goal by end of month:** A working, tested URL-shortener API in Go backed by a well-designed Postgres schema. You can talk fluently about SQL indexing and transactions.

### Week 1 — Go fundamentals + first API skeleton
- **Mon:** Go tour — syntax, structs, slices, maps, errors. Set up your environment + a "hello API" with `net/http`.
- **Tue:** Go interfaces, goroutines & channels (conceptually). Build a `/health` endpoint.
- **Wed:** Project layout, modules, dependency management. Add a router (chi or stdlib).
- **Thu:** Connect to Postgres from Go (`pgx`). Run your first query.
- **Fri:** DSA: arrays & hashing (2 easy problems). Note: "why hashmaps are O(1) average."
- **Sat:** Build `POST /shorten` and `GET /:code` (redirect). Store in Postgres. Generate short codes (base62).
- **Sun:** Tests for the handlers. README with how-to-run. Push.
- **Ship:** A running shortener you can curl.

### Week 2 — SQL deep dive
- **Mon:** Data types, constraints, keys. Design the `links` and `users` tables properly.
- **Tue:** Indexes — B-tree, when they help, when they don't. Add indexes; use `EXPLAIN ANALYZE`.
- **Wed:** Joins in depth (inner/left/anti), and *why* the query planner picks what it does.
- **Thu:** Transactions, isolation levels, deadlocks — the interview-favorite topics.
- **Fri:** DSA: two pointers / sliding window (2 problems). Note isolation levels in your own words.
- **Sat:** Refactor the DB layer: migrations (goose/migrate), parameterized queries, a repository pattern.
- **Sun:** Write 5 "explain this SQL concept" flashcards. Push.
- **Ship:** A clean, migrated schema + notes you could recite in an interview.

### Week 3 — Robust API + testing
- **Mon:** Input validation, error handling, consistent JSON error responses.
- **Tue:** Structured logging (slog). Middleware (logging, recovery).
- **Wed:** Config via env vars, graceful shutdown.
- **Thu:** Unit vs integration tests in Go; table-driven tests.
- **Fri:** DSA: linked lists (2 problems).
- **Sat:** Add integration tests against a real Postgres (testcontainers or a test DB). Add `GET /links` with pagination.
- **Sun:** Coverage check, cleanup, push.
- **Ship:** A properly-tested API. This is already better than a lot of "3-yr" portfolios.

### Week 4 — Caching + the "design TinyURL" story
- **Mon:** Caching concepts: cache-aside, TTL, invalidation, thundering herd.
- **Tue:** Add Redis. Cache the code→URL lookup on redirects.
- **Wed:** Measure it: latency with vs without cache. Record numbers (you'll cite these in interviews).
- **Thu:** Handle collisions and code generation at scale (counter vs random vs hash — tradeoffs).
- **Fri:** DSA: stacks/queues (2 problems).
- **Sat:** Write up "Design TinyURL" as if answering it live — capacity estimation, schema, caching, scaling reads. Base it on *your* implementation.
- **Sun:** Buffer / catch-up day. Push.
- **Ship:** Cache-backed redirects + a written system-design answer grounded in real code.

---

# MONTH 2 — Infra, DevOps, System Design foundations

**Goal by end of month:** Your app is containerized, auto-deployed via CI/CD to the cloud, rate-limited, and you understand the core building blocks of scalable systems.

### Week 5 — Docker
- **Mon:** Images vs containers, layers, Dockerfile basics. Containerize the Go API (multi-stage build).
- **Tue:** docker-compose: API + Postgres + Redis together with one command.
- **Wed:** Volumes, networks, env config, healthchecks.
- **Thu:** Image size optimization; `.dockerignore`; why small images matter.
- **Fri:** DSA: recursion/trees intro (2 problems).
- **Sat:** Full `docker-compose up` brings the whole stack alive. Document it in README.
- **Sun:** Push. Screenshot the running stack.
- **Ship:** Whole app runs anywhere with one command.

### Week 6 — Analytics + rate limiting
- **Mon:** Design the click-event model. Record every redirect (timestamp, referrer, UA, geo-ish).
- **Tue:** Async writes — don't block the redirect on analytics writes. Fire-and-forget with a channel/buffer.
- **Wed:** Rate limiting algorithms: fixed window, sliding window, token bucket. Pick token bucket.
- **Thu:** Implement rate limiting in Redis (per-IP / per-key).
- **Fri:** DSA: binary search + trees (2 problems).
- **Sat:** Build a `/stats/:code` endpoint aggregating clicks. Write the "Design a rate limiter" answer.
- **Sun:** Push, update README with the new architecture diagram.
- **Ship:** Tracked analytics + working rate limiter + two more system-design answers.

### Week 7 — CI/CD + cloud deployment
- **Mon:** CI concepts. Set up GitHub Actions: run tests on every push.
- **Tue:** Add linting, build, and a Docker image build to the pipeline.
- **Wed:** Push image to a registry (GHCR). Understand CD vs CI.
- **Thu:** Deploy to a cheap host (Fly.io, Railway, or a $5 VPS). Get a real URL.
- **Fri:** DSA: hashmap-heavy problems (2). 
- **Sat:** Wire up automated deploy on merge to `main`. Add a managed Postgres.
- **Sun:** Verify the live site. Push.
- **Ship:** A **live, public URL** with automated deploys. This is a massive resume line.

### Week 8 — System design fundamentals
- **Mon:** Load balancing, horizontal vs vertical scaling, statelessness (why your API is already stateless).
- **Tue:** DB scaling: replication, read replicas, sharding basics, when to reach for each.
- **Wed:** Caching layers, CDNs, consistency vs availability (CAP in plain English).
- **Thu:** Queues & async, idempotency, back-pressure.
- **Fri:** DSA: review week — redo 3 problems you found hard.
- **Sat:** Take a classic prompt (e.g., "design a news feed" or "design a rate-limited API gateway") and whiteboard it for 45 min, timed.
- **Sun:** Write the answer up. Push.
- **Ship:** A repeatable mental framework for system-design interviews.

---

# MONTH 3 — Advanced scaling, full-stack polish, interview prep

**Goal by end of month:** A complete, impressive full-stack product + you're interview-ready across system design, coding, and behavioral.

### Week 9 — Message queue + Python worker (your AI/Python slice)
- **Mon:** Why queues: decoupling, spikes, retries. Add Redis Streams / NATS / RabbitMQ.
- **Tue:** Publish click events to the queue instead of writing inline.
- **Wed:** Build a **Python** consumer that processes events (aggregations, and a simple "anomaly/insight" step — this is your AI-adjacent story).
- **Thu:** Idempotency + retry + dead-letter handling.
- **Fri:** DSA: graphs intro (2 problems).
- **Sat:** End-to-end: click → queue → Python worker → analytics store → `/stats`. 
- **Sun:** Push. Update architecture diagram.
- **Ship:** A real event-driven pipeline spanning Go + Python.

### Week 10 — Auth, multi-tenancy, security, observability
- **Mon:** Auth: JWT vs sessions, hashing (bcrypt/argon2). Add signup/login.
- **Tue:** Multi-tenancy: each user sees only their links. Row-level scoping.
- **Wed:** Security basics: OWASP top issues, SQL injection (why you're safe), secrets management.
- **Thu:** Observability: metrics (Prometheus), a dashboard (Grafana), request tracing IDs in logs.
- **Fri:** DSA: graphs BFS/DFS (2 problems).
- **Sat:** Wire monitoring; create one meaningful dashboard (latency, error rate, throughput).
- **Sun:** Push.
- **Ship:** Secure, multi-tenant app you can *observe* in production.

### Week 11 — Angular dashboard + real-time (your home turf)
- **Mon:** Build the dashboard shell in Angular; connect to your API with auth.
- **Tue:** Charts for click analytics (use a charting lib). Make it look genuinely nice.
- **Wed:** Real-time updates via WebSockets/SSE — live click counter.
- **Thu:** Polish UX, loading/error states, responsive layout.
- **Fri:** DSA: dynamic programming intro (1–2 problems — just exposure).
- **Sat:** Deploy the frontend; connect everything live.
- **Sun:** Record a 2-min demo video. Push. Write a great README with the architecture diagram and your metrics.
- **Ship:** A polished, live, full-stack SaaS demo. This is your headline portfolio piece.

### Week 12 — Interview prep sprint + job hunt launch
- **Mon:** System design mocks: do "Design TinyURL," "rate limiter," "analytics" out loud, timed. You've *built* these.
- **Tue:** Behavioral: write STAR stories (a conflict, a failure, a thing you led, a hard bug). 5–6 stories cover most questions.
- **Wed:** Resume rewrite around impact + this project. LinkedIn/GitHub polish.
- **Thu:** DSA: 3 mixed medium problems under time pressure.
- **Fri:** Mock interview (a friend, or record yourself). Fix your weakest area.
- **Sat:** Apply to 15–20 roles (see job-hunt section). Reach out to 3 people for referrals.
- **Sun:** Rest. You earned it.
- **Ship:** Applications out, and you can speak to every topic below.

---

## Interview-readiness checklist (can you answer these cold?)

- Design TinyURL / a URL shortener (capacity, schema, caching, scaling)
- Design a rate limiter (token bucket, distributed state in Redis)
- SQL: difference between indexes types; when an index hurts; read `EXPLAIN` output
- Transactions & isolation levels; what a deadlock is and how to avoid it
- Caching strategies + cache invalidation + thundering herd
- How you'd scale reads vs writes on Postgres
- Docker: image vs container, multi-stage builds, why small images
- Your CI/CD pipeline, step by step, from commit to prod
- Queues: when/why, idempotency, retries, dead-letter queues
- Auth: JWT vs sessions and the tradeoffs
- How you debug a production incident (logs → metrics → traces)
- CAP theorem in plain language with a real example
- A time you disagreed with someone / a failure / something you led (STAR)

If you can answer all of these *and point at code you wrote*, you interview like a senior.

---

## Job hunt & remote strategy (start in Week 11, don't wait until you feel "ready")

- **Where to look for remote:** LinkedIn (filter Remote), We Work Remotely, RemoteOK, Wellfound (AngelList), Toptal (contract, well-paid), and company career pages directly. For higher pay, target product companies and funded startups over agencies.
- **Positioning:** Lead with ".NET + Angular full-stack engineer expanding into Go, cloud-native backends, and system design." Your existing stack is an *asset* — plenty of well-paying .NET/Angular roles exist; you're just adding the senior signals.
- **Resume:** Bullet points are *impact*, not tasks. "Reduced redirect latency 8x by adding a Redis cache layer" beats "worked on caching."
- **Referrals > cold applications.** One warm intro is worth 20 cold applies. Message people kindly, mention the specific role.
- **Apply while unfinished.** Interviews are the best teacher. Start applying at week 11 even if the project isn't perfect.
- **AI tooling as a skill:** mention you use AI coding tools to move faster and review their output critically — that's a senior signal now, not a weakness.

---

## Core resources (don't collect more — these are enough)

- **Go:** "A Tour of Go" → "Learn Go with Tests" (teaches Go *and* testing) → Go by Example.
- **SQL:** "Use The Index, Luke!" (free, excellent on indexing) + practice on your own Postgres.
- **System design:** "System Design Primer" (GitHub, free) + ByteByteGo (YouTube) for visuals.
- **Docker:** the official Docker "Get Started" + one solid YouTube walkthrough.
- **DSA:** NeoCode/LeetCode patterns — focus on *patterns*, not grinding 500 problems.
- **Behavioral:** write your own STAR stories; that beats any book.

---

## Reality checks

- **Falling behind is normal.** The buffer/catch-up days exist on purpose. Ship *something* each week rather than perfectly completing every day.
- **If 3 months of Go feels like too much on top of work,** the fallback is: build the same project in **.NET** (skip Go, keep everything else). You'll lose the "new language" signal but keep 90% of the value and move faster.
- **Depth over breadth.** A single deep project you can discuss for 45 minutes beats five shallow tutorials every time.
