# From Mid → Senior: A 3-Month, Project-Driven Plan

**Your starting point:** 3 yrs Angular + .NET · wants better pay / remote · likes Python & Go
**Your target:** Senior-capable full-stack/backend dev who can pass system-design + practical interviews
**The vehicle:** One project — a URL shortener that grows into an analytics SaaS — that teaches *and* doubles as your interview material and portfolio piece.

---

## 🧭 Coaching Protocol (instructions for Claude — read before responding)

This plan is being worked through with Claude as a **coach, not a code vending machine**. In any conversation where this file is shared, follow this protocol:

**Core rule — don't hand over solutions.** Do not write the learner's feature code, tests, or SQL for them unless they explicitly ask for the solution. Give the map (which stdlib functions/concepts to look up, the shape of the approach, the pitfalls to expect), then let them build it. When they're stuck, ask the question that leads them to the answer rather than giving it on the first time. Small illustrative snippets (a one-line signature, a command to run) are fine; full implementations are not.

**Speed rule — glue vs core (added 2026-11-12, the learner asked to pick up the pace).** Split the work in two:
- **Glue — Claude writes it, with explanations:** YAML (Compose, GitHub Actions), Dockerfiles, Prometheus/Grafana config, Angular scaffolding and UI boilerplate, READMEs, and updates to `week<N>-progress.md`. Write it directly into the repo, explain every non-obvious line, and add questions about it to `interview-qa.md` so the learner can defend it in an interview.
- **Core — the learner writes it:** Go logic that answers an interview question, such as the async click writer, the token-bucket rate limiter, `/stats` aggregation, the queue consumer, idempotency/retry/DLQ, auth and tenant scoping, plus their tests and SQL. The core rule above still applies here.
- Keep momentum: no multi-round discovery on syntax. If it's glue, write it. If it's core, give the map once and let them build.

**Keep the build moving.** Follow the week-by-week plan in order. The learner
learns by building, so most of each reply should be about what to build next.

**Along with telling the learner what to build next give him the pre-requisites what to study before start coding or building.

**When telling to fix a part of code always mention the file name and line number

**Don't block on questions.** If the learner can't answer or wants to move on,
give the answer in a line or two and continue. Don't re-ask. Open questions in
the notes are handled at the end of the entire project.

**Close each topic with the interview answer.** When a day's work is done,
summarise in 3–5 bullets how it answers the matching interview question.

no need to revisit open questions if skipped, these will be handled at the end.

**No side experiments.** Don't assign measurement experiments, benchmarks, or alternative implementations built only to compare approaches. Build time goes to the week's main deliverable. When a plan item is about comparing approaches or tradeoffs, explain it and write the answer into that week's notes instead.

Don't press on things which are not very technical , like writing README.md files which can be written by Ai directly and doesn't test any technical skills

**Review like a senior engineer who's rooting for them.** When reviewing code:
- Name what's genuinely strong, specifically (no generic praise).
- Find real bugs and make them *discover* the bug by tracing the logic (e.g. "trace the expired-link path line by line — what runs after the switch?") rather than just pointing at the fix.
- Flag design smells as tradeoffs to reason about, not just orders.
- Connect each concept to the interview question it answers and to later weeks of this plan.
- If you can't compile/run their code in your environment, say so — your review is by eye.

**Correct misconceptions precisely.** When an answer is almost right, confirm the right part and sharpen the wrong part (e.g. "channels vs mutex", "body cap vs field length cap").

**Tone:** warm, honest. Prose over bullet dumps. One focused next step at the end of each reply.

### Interview Q&A bank — replaces the weekly viva (changed 2026-11-12)
**No weekly vivas.** At the request of the learner, the weekly viva is replaced by a cumulative file, **`interview-qa.md`**, organised by topic, with model answers. After the project, the learner will use it to rehearse with a friend. When a week (or a topic within it) is finished:
- **Add** that week's important interview questions and answers to the matching topic section in `interview-qa.md`. Create a new section if needed, and move items out of "Coming next" (§22).
- Grounded in **their** code and decisions (📍 file references), with *Follow-up* questions an interviewer would ask, and ⭐ on the most common ones.
- Add new bugs found to §20 (STAR material), new numbers to §19, and questions deliberately left open to §21 (unanswered).
- Don't quiz the learner at the end of each week. Keep building. The scoring table below is kept only as a reference for the final mock interview.

### (Reference only) Viva scoring
For the final mock interview, a viva covers: concept questions ("explain in your own words"), "why X over Y" questions, and scenario questions, plus a review of the actual repo/code. Score out of 100:

| Category | Weight | What it measures |
|---|---|---|
| Understanding & explanation | 40 | Can explain it out loud and defend choices under follow-up |
| Working build | 30 | Runs and does what the week required |
| Code craft | 20 | Structure, error handling, idiomatic Go, no footguns |
| Tests & docs | 10 | Someone else can clone and run it; behavior is proven |

Include questions on the glue Claude wrote (Dockerfiles, CI, compose): the learner must be able to explain it line by line.

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
- **Wed:** Finish the cache layer (negative-cache handling, config for TTLs).
- **Thu:** Code generation at scale (counter vs random vs hash). Claude explains the tradeoffs in the notes; nothing extra is built.
- **Fri:** DSA: stacks/queues (2 problems).
- **Sat:** Write up "Design TinyURL" as if answering it live — capacity estimation, schema, caching, scaling reads. Base it on *your* implementation.
- **Sun:** Buffer / catch-up day. Push.
- **Ship:** Cache-backed redirects + a written system-design answer grounded in real code.

---

# MONTH 2 — Infra, DevOps, System Design foundations

**Goal by end of month:** Your app is containerized, auto-deployed via CI/CD to the cloud, rate-limited, and you understand the core building blocks of scalable systems.

> **⚡ Accelerated plan (from 2026-11-12, Week 5 Thu onward).** Weeks 5–12 below were compressed at the learner's request: roughly 7.5 remaining weeks → about 4.5–5. A "week" is now a **sprint of sessions**, not 7 calendar days. Move to the next week as soon as the build works and `interview-qa.md` is updated. Week numbers are unchanged so `week<N>-*.md` file names stay consistent. Week 8 no longer exists as a standalone week (its topics are folded into Weeks 7, 9 and 12). Glue vs core split: see the Speed rule in the Coaching Protocol. DSA stays at 2 problems per week but never blocks build progress.

### Week 5 — Docker
- **Mon:** ✅ Images vs containers, layers, Dockerfile basics. Containerize the Go API (multi-stage build).
- **Tue:** ✅ docker-compose: API + Postgres + Redis together with one command.
- **Wed:** ✅ Volumes, networks, env config, healthchecks.
- **Thu (1 session):** Image size: `-ldflags="-s -w"` + switch the runtime base to distroless. Scratch vs distroless vs alpine tradeoffs go in the notes (no side-by-side builds).
- **Fri:** DSA: recursion/trees intro (2 problems).
- **Sat → compressed:** Claude writes the README "run with Docker" section and adds Week 5 Thu to `interview-qa.md`.
- **Ship:** Whole app runs anywhere with one command.

### Week 6 — Analytics + rate limiting (full depth, core = learner)
- **S1:** Design the click-event model + migration. Record every redirect (timestamp, referrer, UA). ~~geo-ish~~ cut.
- **S2:** Async writes: don't block the redirect on analytics writes. Buffered channel + background writer, with graceful-shutdown drain.
- **S3:** Rate limiting algorithms (fixed window, sliding window, token bucket) explained in notes → implement token bucket in Redis (per-IP), as middleware.
- **S4:** `/stats/:code` endpoint aggregating clicks. Write the "Design a rate limiter" answer.
- **Fri:** DSA: binary search + trees (2 problems).
- Claude updates the README, the architecture diagram and `interview-qa.md`.
- **Ship:** Tracked analytics + working rate limiter + two more system-design answers.

### Week 7 — CI/CD + cloud deployment (glue = Claude, learner explains every step)
- **S1:** One GitHub Actions pipeline: test → lint → build → Docker image → push to GHCR. Claude writes the workflow; learner learns CI vs CD and can walk through it step by step.
- **S2:** Deploy to a cheap host (Fly.io or Railway) with managed Postgres + Redis, migrations as a release step, and automatic deploy on merge to `main`. Get a real URL.
- **Notes (folded from old Week 8):** load balancing, horizontal vs vertical scaling, why the API is stateless, runtime restart/recovery on the host.
- **Fri:** DSA: hashmap-heavy problems (2).
- **Ship:** A **live, public URL** with automated deploys. This is a massive resume line.

### ~~Week 8 — System design fundamentals~~ → folded into other weeks
No standalone week. Where each topic went:
- Load balancing, scaling, statelessness → Week 7 notes.
- Queues, idempotency, back-pressure → Week 9 (built there).
- DB scaling (replication, read replicas, sharding), caching layers/CDNs, CAP → Week 12 Mon notes, tied to Snip's actual design.
- The 45-min timed whiteboard of a *new* prompt (news feed / API gateway) → Week 12 Mon.

---

# MONTH 3 — Advanced scaling, full-stack polish, interview prep

**Goal by end of month:** A complete, impressive full-stack product + you're interview-ready across system design, coding, and behavioral.

### Week 9 — Message queue + Python worker (your AI/Python slice) — **start applying to jobs this week**
- **S1:** Why queues (decoupling, spikes, retries; back-pressure in notes). Use **Redis Streams**, which is already in the stack, so no new infra. Publish click events to the stream instead of writing inline.
- **S2:** **Python** consumer with consumer groups: aggregations into the analytics store. Claude writes the Python project scaffolding/Dockerfile; learner writes the consume/ack logic.
- **S3:** Idempotency + retry + dead-letter handling. End-to-end: click → stream → Python worker → analytics store → `/stats`.
- "Anomaly/insight" step: **optional**, 1 hour max, only if time allows.
- **Fri:** DSA: graphs intro (2 problems).
- **Ship:** A real event-driven pipeline spanning Go + Python.

### Week 10 — Auth, multi-tenancy, observability
- **S1:** Auth: JWT vs sessions (notes), bcrypt. Signup/login + auth middleware.
- **S2:** Multi-tenancy: each user sees only their links (scoping in the repository layer). Security basics (OWASP, why parameterized queries make SQL injection impossible here, secrets management) → **notes only**.
- **S3:** Observability: request IDs in slog logs (learner), Prometheus `/metrics` with latency/error/throughput (learner instruments), Grafana dashboard + compose wiring (Claude configures).
- **Fri:** DSA: graphs BFS/DFS (2 problems).
- **Ship:** Secure, multi-tenant app you can *observe* in production.

### Week 11 — Angular dashboard (your home turf, so compressed hardest)
- **S1:** Claude scaffolds the Angular app (login, links list, stats chart with a charting lib); learner wires it to the API with auth and reviews it.
- **S2:** Deploy the frontend; connect everything live. Claude writes the final README with the architecture diagram and metrics.
- Optional if time allows: SSE live click counter, 2-min demo video.
- **Fri:** DSA: dynamic programming intro (1–2 problems, just exposure).
- **Ship:** A live, full-stack SaaS demo. This is your headline portfolio piece.

### Week 12 — Interview prep sprint + job hunt
- **Mon:** System design mocks: "Design TinyURL", "rate limiter", "analytics" out loud and timed (you've *built* these), plus one 45-min whiteboard of a new prompt. Notes on DB scaling (replicas, sharding), CDNs and CAP (folded from old Week 8).
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

## Job hunt & remote strategy (start in Week 9, don't wait until you feel "ready")

- **Where to look for remote:** LinkedIn (filter Remote), We Work Remotely, RemoteOK, Wellfound (AngelList), Toptal (contract, well-paid), and company career pages directly. For higher pay, target product companies and funded startups over agencies.
- **Positioning:** Lead with ".NET + Angular full-stack engineer expanding into Go, cloud-native backends, and system design." Your existing stack is an *asset* — plenty of well-paying .NET/Angular roles exist; you're just adding the senior signals.
- **Resume:** Bullet points are *impact*, not tasks. "Reduced redirect latency 8x by adding a Redis cache layer" beats "worked on caching."
- **Referrals > cold applications.** One warm intro is worth 20 cold applies. Message people kindly, mention the specific role.
- **Apply while unfinished.** Interviews are the best teacher. Start applying at week 9 even if the project isn't perfect.
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
