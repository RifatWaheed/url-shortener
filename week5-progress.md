# Week 5 Progress — Docker

**Where I stopped (2026-11-10):** Week 5 **Monday is DONE** (app runs in a container, all checks passed, graceful shutdown verified). **Next is Tuesday: `docker-compose.yml`** (see "After Monday" below).

---

## Done so far

| What | Result |
|---|---|
| Docker Desktop installed (WSL 2 backend) | `docker version` shows Client + Server |
| `.dockerignore` | Excludes `.env`, `*.exe`, `.git`, `.env.docker`. Verified `.env` was NOT copied into the image |
| `Dockerfile` stage 1 (build) | Compiles a static binary at `/bin/snip` (18.7 MB) |
| `Dockerfile` stage 2 (runtime) | Copies only the binary into `alpine:3.24` |
| Image size | **804 MB → 41 MB** (13.1 MB compressed) after adding stage 2 |
| Checked inside final image | `/app` doesn't exist (source, tests, notes are left behind); `/bin/snip` is there |

## Decisions made (and why)

- **Builder `golang:1.27-alpine`**: the Go in the image must be **at least** the `go` version in go.mod (1.26.6). Never use `tip` (unreleased dev Go).
- **`WORKDIR /app`**: every later `RUN`/`COPY` runs in that folder. My first attempt didn't have it, so `go mod download` couldn't find go.mod.
- **Copy go.mod/go.sum → `go mod download` → then `COPY . .`**: layer caching. Editing a `.go` file doesn't re-download libraries.
- **`CGO_ENABLED=0`**: a fully self-contained binary that needs no C library at runtime, so it can run on tiny or empty base images. All my deps (pgx, go-redis, godotenv) are pure Go.
- **Multi-stage**: stage 1 is the "kitchen" (Go toolchain, source); stage 2 is the "box" (just the binary). Only the last stage becomes the image.
- **Runtime base `alpine:3.24`**: small, but still has a shell and `ls` for debugging. `scratch`/distroless is revisited on **Thursday** (smaller, but no shell; scratch also has no CA certs, which matters for TLS to managed Postgres in Week 7).
- **Non-root user (`adduser -D -u 10001 appuser` + `USER appuser`)**: if the app is ever exploited, the attacker isn't root.
- **`ENTRYPOINT ["/bin/snip"]` (exec form, brackets)**: `docker stop` sends SIGTERM straight to my program, which handles it at `main.go:31` (graceful shutdown). In shell form, a shell sits in the middle and doesn't pass the signal on, so Docker force-kills after 10 s.
- **Config comes from env vars, not `.env`, inside containers.** The warning "unable to find env variables" at startup is expected.

---

## Setting up the home PC first

1. `git pull`
2. Install **Docker Desktop** (WSL 2 backend) if it isn't installed. Open it and wait for **"Engine running"**. Check with `docker version`: there must be a **Server:** section.
3. Make sure **Postgres** is running at home and the `url_shortener` database has its tables. If it's a fresh database, run the migrations (see README):
   ```powershell
   migrate -path migrations -database "postgres://postgres:YOUR_PASSWORD@localhost:5432/url_shortener?sslmode=disable" up
   ```
4. Rebuild the image (images don't travel through git; only the Dockerfile does):
   ```powershell
   docker build -t snip-api .
   docker images snip-api        # expect ~41 MB
   ```

---

## Continue here: run the app inside the container

### The key idea: what "localhost" means

**Every container is its own little computer.** `localhost` / `127.0.0.1` always means "**me**, this same computer."

- Inside the container, `localhost:5432` looks for Postgres **inside the container**, where it doesn't exist. Use **`host.docker.internal`**, Docker Desktop's name for "the Windows machine I'm running on."
- **Bug in the other direction, `main.go:51`:** `ADDR` defaults to `127.0.0.1:8080`, meaning "only accept visitors from inside this same computer." Requests from Windows come in from *outside* the container through Docker's port forwarding, so the app would ignore them. Listen on **`0.0.0.0:8080`** ("accept visitors from anywhere"). Fix it through config (the env var), not code; `127.0.0.1` stays the safe default for laptop runs.

| Setting | On Windows (`.env`) | Inside the container (`.env.docker`) |
|---|---|---|
| Postgres host | `localhost` | `host.docker.internal` |
| Redis address | `127.0.0.1:6379` | `host.docker.internal:6379` |
| Listen address | `127.0.0.1:8080` | `0.0.0.0:8080` |
| BASE_URL | `http://127.0.0.1:8080` | `http://localhost:8080` (what the *user* on Windows sees) |

### Step 1: Start Redis in a container

```powershell
docker run -d --name snip-redis -p 6379:6379 redis:8-alpine
docker ps          # snip-redis should show "Up"
```
- `-d` runs it in the background · `--name` names it · `-p WINDOWS_PORT:CONTAINER_PORT` · `redis:8-alpine` is the image.
- If you get "name already in use" (e.g. after a restart): `docker start snip-redis`.

### Step 2: Create `.env.docker` in the project root

It's already in `.gitignore` and `.dockerignore`, so it never gets committed or baked into the image. Create it again on each machine:

```
DATABASE_URL=postgres://postgres:YOUR_PASSWORD@host.docker.internal:5432/url_shortener?sslmode=disable
REDIS_ADDR=host.docker.internal:6379
ADDR=0.0.0.0:8080
BASE_URL=http://localhost:8080
```
Rules: **no quotes**, **no spaces around `=`**.

### Step 3: Run the app container

```powershell
docker run --rm --name snip-api -p 8080:8080 --env-file .env.docker snip-api
```
- `--rm` deletes the container when it stops (the image stays) · `-p 8080:8080` maps Windows port 8080 to the container's 8080 · `--env-file` loads settings.

**Expected logs (JSON):**
1. warning: unable to find env variables (**expected**: there's no `.env` in the image on purpose)
2. `connected to database`
3. `connected to redis`
4. `listening on ... 0.0.0.0:8080`

If you see `pg_hba.conf` or `password authentication failed`: that's a Postgres permission setting. Paste the exact error into the next session.

### Step 4: Test from a **second** PowerShell window

```powershell
curl.exe http://localhost:8080/health
```
```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/shorten -ContentType 'application/json' -Body '{"url":"https://example.com"}'
```
```powershell
curl.exe -i http://localhost:8080/PUT_CODE_HERE     # expect 302 + Location: https://example.com
```
(Use `curl.exe`, not `curl`. In PowerShell 5.1, `curl` is an alias for `Invoke-WebRequest`.) The first window should log each request.

### Step 5: Test graceful shutdown

```powershell
docker stop snip-api
```
In the first window you want **`shutdown signal received`** then **`server stopped cleanly`**, almost instantly. If it hangs about 10 s and dies without those lines, the signal isn't reaching the program. Bring that to the next session.

---

## Monday is done when

- [x] Container logs show connected to database, connected to redis, listening on 0.0.0.0:8080
- [x] `/health`, `POST /shorten` and the redirect all work from Windows
- [x] `docker stop` produces "server stopped cleanly" almost instantly
- [x] Results of the above reported back to Claude (logs or errors)

## After Monday

- **Tuesday:** `docker-compose.yml` brings up API + Postgres + Redis with one command (replaces the long `docker run` commands; `env_file:` replaces `--env-file`; containers reach each other by **service name** instead of `host.docker.internal`).
- **Wednesday:** volumes, networks, healthchecks.
- **Thursday:** image size, `scratch` vs distroless vs alpine, `-ldflags="-s -w"`.
- `week5-notes.md` is written (build/tags, layers, why image IDs change, build cache, multi-stage, `docker run` flags, localhost-inside-a-container, exec form). Study it before the Week 5 viva.

## Useful commands

| Command | What it does |
|---|---|
| `docker ps` / `docker ps -a` | Running containers / all containers |
| `docker images` | Images on this machine |
| `docker logs snip-redis` | Logs of a background container |
| `docker stop <name>` / `docker start <name>` | Stop / start a container |
| `docker rm <name>` | Delete a stopped container |
| `docker run --rm --entrypoint ls snip-api -la /bin` | Look inside the image without starting the app |
