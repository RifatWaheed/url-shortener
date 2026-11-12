# Week 5 Progress — Docker

**Where I stopped (2026-11-12):** Week 5 **Monday, Tuesday and Wednesday are DONE**. `docker compose up` brings up the whole stack in the right order, with **zero** `connection refused` on a cold start, and the Postgres data survives `down`/`up`. **Next is Thursday: image size** (see "Continue here").

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

Verified: `\dt` shows 4 tables + `schema_migrations`; rerunning `migrate` prints `no change`; `/health`, `POST /shorten` and the redirect work from Windows; the link lands in the **container's** Postgres. `.env.docker` taken out of git (it held my real Windows Postgres password).

Tuesday exposed two gaps: **no persistence** (data gone after `down`/`up`) and a **startup race** (API crashed 4× with `connection refused` while Postgres booted; `restart: on-failure` hid it).

### Wednesday: volumes, healthchecks, startup order

| What | Where in `docker-compose.yml` | Result |
|---|---|---|
| Named volume `pgdata` mounted at `/var/lib/postgresql` (PG 18 path) | lines 3-4, declared at lines 57-58 | Link `tJAfZvy` survived `down` → `up`. `docker volume ls` still shows `url-shortener_pgdata` |
| Postgres healthcheck: `pg_isready -h localhost -U postgres -d url_shortener` | lines 12-17 | `(healthy)` after ~6 s. `-h localhost` forces TCP so it can't pass during the init-phase temporary server |
| Redis healthcheck: `redis-cli ping` | lines 21-26 | `(healthy)` |
| `migrate` waits for `postgres: service_healthy` | lines 34-36 | Never races Postgres |
| `api` waits for `postgres` + `redis` healthy and `migrate: service_completed_successfully` | lines 48-54 | API sits in `Created` ~7 s, then connects on the **first try** |
| Removed `restart: on-failure` from `migrate` and `api` | — | Test is honest; a failed migration now blocks the API instead of looping |

Cold-start test (`down -v` → `up --build -d`):
- **Before:** `api` started 1.3 s in, 3× `connection refused` to `172.18.0.3:5432`, `Restarting (1)`.
- **After:** `migrate` → `Exited (0)` (`docker inspect ... {{.State.ExitCode}}` = `0`), postgres/redis `(healthy)`, API logs `connected to database` → `connected to redis` → `listening on` with no errors.

Networks: kept the single default network (nothing built). When to split is written in notes §19.

Side findings:
- Two hex-named volumes in `docker volume ls` are **anonymous volumes**, most likely from the Redis image's `VOLUME /data`. `docker volume prune` cleans unused ones.
- `Invoke-RestMethod` on `/shorten` showed blank `shortCode`/`shortUrl` columns. That's a PowerShell 5.1 display quirk: curl showed the correct `201` body. Use `| ConvertTo-Json` to see the data.

## Decisions made (and why)

- **Builder `golang:1.27-alpine`**: must be at least the `go` version in go.mod.
- **Copy go.mod/go.sum → `go mod download` → `COPY . .`**: layer caching (stable first, changing last).
- **`CGO_ENABLED=0`**: static binary, runs on tiny bases.
- **Non-root user, exec-form `ENTRYPOINT`**: limits damage if exploited; SIGTERM reaches the app for graceful shutdown.
- **Pinned major versions** (`postgres:18-alpine`, `redis:8-alpine`), `migrate` pinned to an exact release, plain multi-arch tags.
- **Only the API is published.** Postgres on `127.0.0.1` for pgAdmin; Redis not published (no auth, poisoned cache = hijacked redirects).
- **Migrations as a separate one-shot service**, not `/docker-entrypoint-initdb.d` (it would run `.down.sql` files and only runs on an empty data folder).
- **Dev-only throwaway password (`snip_dev`) in the compose file.** Real Windows password stays out of git.
- **Config in `environment:`**, not in the image. The same `snip-api` image runs everywhere.
- **Named volume, not a bind mount, for Postgres data**: Docker owns the files (correct permissions, no Windows path issues); bind mounts stay for things I edit (`./migrations`).
- **Mount `/var/lib/postgresql`, not `/data`**: required by PG 18+; leaves room for a side-by-side major upgrade.
- **Healthcheck + `depends_on` conditions instead of `restart: on-failure`**: ordering is explicit; misconfiguration fails loudly once instead of crash-looping.
- **Migrations are a gate** (`service_completed_successfully`): the API never starts against a half-migrated schema.
- **No restart policy for now.** Runtime crash recovery is a separate concern; it moves to the cloud host in Week 7 (or `restart: unless-stopped` in prod Compose).

---

## Setting up the home PC

Windows Postgres and `.env.docker` are **not needed** for the Docker stack.

1. `git pull`
2. Docker Desktop open, showing **"Engine running"**.
3. ```powershell
   docker compose up --build -d
   docker compose ps -a
   docker compose logs api
   ```
4. Test with `curl.exe http://localhost:8080/health`.

The home PC gets its own empty `pgdata` volume on the first `up` (volumes are per machine, not in git).

---

## Continue here: Thursday (image size)

Read first:
- `-ldflags="-s -w"`: what the symbol table and DWARF debug info are, and what you lose by stripping them.
- Base images `scratch` vs distroless (`gcr.io/distroless/static`) vs `alpine`: what's inside each, and what you can no longer do (no shell for `docker exec`, no `wget` for a healthcheck, CA certificates, timezone data, how to run as non-root without `adduser`).

Build:
1. Add `-ldflags="-s -w"` to the `go build` in `Dockerfile` and compare the size against the current 41 MB.
2. Pick a runtime base and justify the tradeoff in notes.
3. Rerun the cold-start test to confirm the stack still comes up healthy.

Then **Saturday**: full `docker compose up` documented in README.

---

## Useful commands

| Command | What it does |
|---|---|
| `docker compose config` | Validate the file and print it fully expanded (short forms shown in long form) |
| `docker compose up --build -d` | Build the API image and start everything in the background (waits on healthchecks) |
| `docker compose up migrate` | Run one service in the foreground and see its output |
| `docker compose ps` / `ps -a` | Running services / including exited ones (like `migrate`). STATUS shows `(healthy)` |
| `docker compose logs api` / `logs -f api` | API logs / keep following them (Ctrl+C stops following) |
| `docker compose exec redis redis-cli ping` | Run a command inside a running container |
| `echo $LASTEXITCODE` | Exit code of the last program in PowerShell (what Docker sees from a healthcheck) |
| `docker compose exec postgres psql -U postgres -d url_shortener -P pager=off -c "\dt"` | SQL inside the container's Postgres (`-P pager=off` stops `--More--`) |
| `docker inspect url-shortener-migrate-1 --format "{{.State.ExitCode}}"` | Exit code of a finished container |
| `docker compose down` | Stop and **remove** containers + network. **Keeps volumes** |
| `docker compose down -v` | Same, **and deletes volumes** (true cold start, data gone) |
| `docker volume ls` / `docker volume prune` | List volumes / remove unused ones (anonymous leftovers) |
| `docker images` / `docker ps -a` | Images / all containers on this machine |
