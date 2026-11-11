# Week 5 Progress — Docker

**Where I stopped (2026-11-11):** Week 5 **Monday and Tuesday are DONE**. `docker compose up` brings up the whole stack (API + Postgres + Redis + migrations). **Next is Wednesday: volumes, networks, healthchecks.** It fixes the two gaps found on Tuesday (see "Continue here").

---

## Done so far

### Monday: the app in a container

| What | Result |
|---|---|
| Docker Desktop installed (WSL 2 backend) | `docker version` shows Client + Server |
| `.dockerignore` | Excludes `.env`, `*.exe`, `*.md`, `.git`, `.env.docker`. Verified `.env` was NOT copied into the image |
| `Dockerfile` stage 1 (build) | Compiles a static binary at `/bin/snip` |
| `Dockerfile` stage 2 (runtime) | `alpine:3.24` + non-root user + only the binary |
| Image size | **804 MB → 41 MB** (13.1 MB compressed) |
| `docker run` against Windows Postgres + a Redis container | Works through `host.docker.internal`; graceful shutdown verified |

### Tuesday: `docker-compose.yml`

| Service | Image | What it does | Published? |
|---|---|---|---|
| `postgres` | `postgres:18-alpine` | Database. `POSTGRES_USER/PASSWORD/DB` create the user and the `url_shortener` DB on first start | `127.0.0.1:5433→5432` (pgAdmin only, this PC only) |
| `redis` | `redis:8-alpine` | Cache | **No** (hallway only) |
| `migrate` | `migrate/migrate:v4.20.1` | One-shot job: runs `migrations/` against Postgres, then exits 0 | No |
| `api` | `snip-api` (`build: .`) | My Go app | `8080→8080` |

Verified:
- `docker compose ps` shows postgres/redis running; `migrate` shows `Exited (0)` in `ps -a`.
- `\dt` shows 4 tables + `schema_migrations`. Running `migrate` a second time prints `no change` (idempotent).
- API logs: connected to database, connected to redis, listening on `0.0.0.0:8080`.
- `/health`, `POST /shorten` and the redirect (`302` + `Location`) work from Windows.
- The new link is in the **container's** Postgres, not the Windows one (checked in pgAdmin).
- Took `.env.docker` out of git (`.gitignore` line 3 + `git rm --cached`). It held my real Windows Postgres password.

### Two gaps found on Tuesday (Wednesday fixes them)

1. **No persistence.** After `docker compose down` + `up`, the `links` table was empty (`0 rows`). The data lived inside the Postgres container and was removed with it. Fix: a **named volume**.
2. **Startup race.** From a cold start the API crashed 4 times with `connection refused` (Postgres still booting) before connecting about 3 s later. `depends_on` only waits for the container to *start*, and `main.go:59-63` pings once and exits. `restart: on-failure` hides it, but a wrong password would crash-loop forever. Fix: a **healthcheck** + `depends_on: condition: service_healthy`.

## Decisions made (and why)

- **Builder `golang:1.27-alpine`**: must be at least the `go` version in go.mod.
- **Copy go.mod/go.sum → `go mod download` → `COPY . .`**: layer caching (stable first, changing last).
- **`CGO_ENABLED=0`**: static binary, runs on tiny bases.
- **Non-root user, exec-form `ENTRYPOINT`**: limits damage if exploited; SIGTERM reaches the app for graceful shutdown.
- **Pinned major versions** (`postgres:18-alpine`, `redis:8-alpine`): get patch updates, never an accidental major jump. `migrate` is pinned to an exact release, `v4.20.1`.
- **Plain tag, not `-amd64`/`-arm64`**: the plain tag is a manifest list, so Docker picks the right CPU variant.
- **Only the API is published.** Postgres is bound to `127.0.0.1` for pgAdmin. Redis isn't published: no auth by default, and anyone who can write to it can poison the cache and redirect my users.
- **Migrations as a separate one-shot service**, not in Postgres's `/docker-entrypoint-initdb.d`. initdb runs every `.sql` file alphabetically, so the `.down.sql` files would run too, and it only runs on an empty data folder.
- **Dev-only throwaway password (`snip_dev`) written in the compose file.** It's acceptable because the DB only exists inside Docker on my laptop. My real Windows password stays out of git.
- **Config in `environment:`**, not in the image. The same `snip-api` image runs everywhere.
- **`restart: on-failure`** is a temporary patch for the race, to be replaced Wednesday.

---

## Setting up the home PC

Much simpler now. Windows Postgres and `.env.docker` are **not needed** for the Docker stack.

1. `git pull`
2. Docker Desktop open, showing **"Engine running"**.
3. ```powershell
   docker compose up --build -d
   docker compose ps -a
   docker compose logs api
   ```
4. Test with `curl.exe http://localhost:8080/health`.

---

## Continue here: Wednesday (volumes, networks, healthchecks)

1. **Named volume for Postgres** so data survives `down`/`up`. Then repeat Tuesday's experiment and the link should still be there. Learn `docker compose down` vs `down -v`. Note: from version 18, the Postgres image keeps its data under `/var/lib/postgresql`. Check the Docker Hub page for the exact path to mount.
2. **Healthcheck on Postgres** (`pg_isready`) and Redis (`redis-cli ping`).
3. **`depends_on` with conditions**: `migrate` waits for `service_healthy` Postgres; `api` waits for `service_completed_successfully` migrate + healthy Postgres/Redis. Then try removing `restart: on-failure` and confirm the cold start shows **zero** `connection refused` errors.
4. **Networks**: what the default network gives you, and when you'd split into several networks.

Then **Thursday**: image size (`scratch` vs distroless vs alpine, `-ldflags="-s -w"`).

---

## Useful commands

| Command | What it does |
|---|---|
| `docker compose config` | Validate the file and print it fully expanded |
| `docker compose up --build -d` | Build the API image and start everything in the background |
| `docker compose up migrate` | Run one service in the foreground and see its output |
| `docker compose ps` / `ps -a` | Running services / including exited ones (like `migrate`) |
| `docker compose logs api` / `logs -f api` | API logs / keep following them (Ctrl+C stops following) |
| `docker compose exec redis redis-cli ping` | Run a command inside a running container |
| `docker compose exec postgres psql -U postgres -d url_shortener -P pager=off -c "\dt"` | SQL inside the container's Postgres (`-P pager=off` stops `--More--`) |
| `docker compose down` | Stop and **remove** containers + network |
| `docker images` / `docker ps -a` | Images / all containers on this machine |
