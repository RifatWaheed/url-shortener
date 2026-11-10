# Week 5 Notes — Docker

---

## 1. Images, containers, and `docker build`

**Q: What's the difference between an image and a container?**
Ans: An image is a read-only template: a stack of filesystem layers plus a config (entrypoint, user, exposed ports, env). A container is a running (or stopped) instance of an image. It's the image plus a thin writable layer on top, and a process. One image can start many containers. `docker build` makes images; `docker run` makes containers.

Follow-up: *If I change a file inside a running container, does the image change?*
Ans: No. The change goes into that container's writable layer only. The image is never modified. With `--rm`, that writable layer is thrown away when the container stops.

**Q: Explain `docker build -t snip-api .`**
Ans:
- `docker build` reads the Dockerfile and runs its instructions in order to produce an image.
- `-t snip-api` (short for `--tag`) names the image. With no `:tag`, Docker adds `:latest`, so the full name is `snip-api:latest`.
- `.` is the **build context**: the directory sent to the Docker engine. The build can only see files inside it. It isn't "where the Dockerfile is", although Docker looks for `Dockerfile` there by default.

Follow-up: *What does `docker build -t snip-api . .` do?*
Ans: It fails. `docker build` takes exactly one positional argument (the context).

**Q: In `COPY . .`, what do the two dots mean?**
Ans: The first `.` is the build context ("everything I was sent"). The second `.` is the current `WORKDIR` inside the image (`/app`). So it means "copy the whole context into `/app`."

Follow-up: *Can I `COPY ../secrets.txt .`?*
Ans: No. Files outside the context are never sent to the engine, so the build can't reach them.

---

## 2. Tags

**Q: What is a tag?**
Ans: A full image reference is `name:tag`, e.g. `snip-api:v1`. The **name** (Docker calls it the repository) says *which image*; the **tag** says *which version*. A tag is a **mutable pointer** to an image, much like a git branch name points to a commit.

**Q: Does `latest` mean the newest build?**
Ans: No. It's just the default label when you don't write a tag. Nothing keeps it pointing at your most recent build. That's why CI tags images with something meaningful like the git commit SHA (`snip-api:df61e56`): you can always say exactly what's running in prod, and roll back by redeploying an older tag (Week 7).

**Q: What happens if I build `snip-api:v1` again?**
Ans: The `v1` tag **moves** to the new image. The old image loses its name and becomes *dangling* (untagged). `docker image prune` removes dangling images.

**Q: Can one image have several tags?**
Ans: Yes. `docker build -t snip-api:latest -t snip-api:v1 .` puts both tags on the same image. `docker rmi snip-api:v1` then only removes that tag. The layers are deleted only when nothing references them anymore.

---

## 3. Layers and content addressing

**Q: What is a layer?**
Ans: The filesystem change produced by one Dockerfile instruction that changes files (`FROM`, `COPY`, `RUN`). Each layer is identified by a **hash of its contents**. Identical contents give an identical layer, no matter when or how it was built, so it's stored once and shared between images.

**Q: My final image has 3 layers. Where do they come from?**
Ans:

| Layer | Instruction |
|---|---|
| 1 | `FROM alpine:3.24` (the base) |
| 2 | `COPY --from=build /bin/snip /bin/snip` (my binary) |
| 3 | `RUN adduser -D -u 10001 appuser` |

`USER`, `EXPOSE` and `ENTRYPOINT` only change the image **config** (metadata), so they add no layer. Nothing from the `build` stage (Go toolchain, source, module cache) is in the final image.

**Q: `snip-api:latest` and `snip-api:v1` each show 41 MB. Do they use 82 MB together?**
Ans: No. Shared layers are counted in each row of `docker images` but stored once on disk. Ten versions of an app on the same base don't cost ten times the space.

**Q: DISK USAGE 41 MB vs CONTENT SIZE 13.1 MB?**
Ans: Content size is the **compressed** size, roughly what gets pushed to and pulled from a registry. Disk usage is the **unpacked** size on disk.

**Q: How do I check whether two images really have the same contents?**
Ans: Compare their layer lists, not their IDs:
```
docker image inspect snip-api:latest --format "{{json .RootFS.Layers}}"
docker image inspect snip-api:v1     --format "{{json .RootFS.Layers}}"
```
Identical lists mean an identical filesystem.

---

## 4. Why every build gets a new image ID

**Q: I rebuilt `v1` several times without changing a single file. Every build had a new image ID. Why?**
Ans: The ID Docker Desktop shows is the digest of the **manifest list** (an index), not of the layers. BuildKit puts two things in that index:
1. the actual image manifest (layers + config), and
2. an **attestation manifest**, which is *provenance* metadata describing how and when the image was built.

The provenance includes build-time details that change on every build. The index's digest is a hash of everything inside it, so it changes on every build too, even when every layer is byte-for-byte identical. You can see it in the build log: the `exporting manifest list sha256:8d9dd6...` line matches the image ID that `docker images` shows.

Follow-up: *So does a different image ID mean different contents?*
Ans: Not necessarily. "Different ID" only means the wrapper changed. Compare `RootFS.Layers` to know whether the filesystem changed.

Follow-up: *Can I get a stable ID?*
Ans: Building with `--provenance=false` leaves out the attestation, so a fully cached rebuild usually keeps the same ID. In practice it doesn't matter: tags and commit SHAs identify versions, not image IDs.

---

## 5. Build cache

**Q: How does the build cache decide whether to reuse a step?**
Ans: Steps are checked top to bottom. A step is reused if the instruction is the same, the layer before it is the same, and (for `COPY`) the copied files have the same contents. **Once one step misses, every step after it in that stage re-runs**, because each layer is built on top of the one before.

**Q: Why copy `go.mod`/`go.sum` and run `go mod download` before `COPY . .`?**
Ans: Dependencies change rarely; code changes all the time. With this order, editing a `.go` file only invalidates `COPY . .` and the steps after it. The `go mod download` layer stays cached, so libraries aren't downloaded again on every build.

**Q: In one build, `COPY . .` and `go build` re-ran (18.5 s), but `COPY --from=build /bin/snip` was CACHED. How?**
Ans: Something in the build context changed, which invalidated `COPY . .` and forced a recompile. But the compile produced a byte-for-byte identical binary, because Go builds are reproducible: same source, same binary. `COPY --from` compares file contents, so its layer matched and the rest of the final stage was reused.

Follow-up: *What could have changed if the binary didn't?*
Ans: A file that isn't Go code but is still in the context, such as a `week5-*.md` note, the README, or the Dockerfile itself. `COPY . .` copies them all, so editing any of them throws away the cache for `COPY . .` and costs a full recompile for nothing.

Follow-up: *How do you stop that?*
Ans: Add those files to `.dockerignore`. Ignored files are never sent in the context, so they can't break the cache, and they can't leak into the build stage (e.g. `.env` secrets). Mine currently ignores `.env`, `*.exe`, `.git`, `.env.docker`. Notes and docs are candidates to add.

**Q: Lines 11–12 of the Dockerfile are `COPY --from=build ...` then `RUN adduser ...`. Better order?**
Ans: Put `RUN adduser ...` **before** the `COPY --from=build`. Creating the user doesn't depend on my code, so it should sit above the layer that changes on every code change. Then a code change only rebuilds the binary layer. It's a small win, but it's the same rule as `go mod download`: **stable things first, things that change often last.**

---

## 6. Multi-stage builds

**Q: What does a multi-stage build do, and why use one?**
Ans: Stage 1 (`golang:1.27-alpine AS build`) is the kitchen: it has the Go toolchain and source and compiles the binary. Stage 2 (`alpine:3.24`) is the box: it copies **only** the binary from stage 1. Only the last stage becomes the image. Mine went from **804 MB to 41 MB** (13.1 MB compressed).

Follow-up: *Why do small images matter?*
Ans: Faster pushes, pulls and deploys (and faster scale-out when new servers must pull the image), less disk, and a smaller attack surface: no compiler, no source code, fewer packages with possible vulnerabilities.

**Q: Why `CGO_ENABLED=0`?**
Ans: It produces a fully static binary that needs no C library at runtime, so it runs on tiny bases like alpine or even `scratch`. All my dependencies are pure Go, so nothing is lost.

---

## 7. `docker run` flags

**Q: What do these flags do?**

| Flag | Meaning | Example |
|---|---|---|
| `-p HOST:CONTAINER` | **Publish a port.** Traffic to the host port is forwarded to the container port. Without it, nothing outside Docker can reach the app. | `-p 8080:8080` |
| `--rm` | Delete the container automatically when it stops. The image stays. Good for throwaway runs. | `--rm` |
| `-e KEY=VALUE` | Set one environment variable in the container. | `-e ADDR=0.0.0.0:8080` |
| `--env-file FILE` | Load many env vars from a file (`KEY=VALUE` per line, no quotes, no spaces around `=`). | `--env-file .env.docker` |
| `-d` | Detached: run in the background and give the terminal back. Read output with `docker logs <name>`. | `-d` |
| `--name NAME` | Give the container a fixed name instead of a random one, so you can `docker stop/start/logs NAME`. | `--name snip-api` |
| `-it` | Interactive + terminal: keep stdin open and attach a TTY. Used to get a shell. | `docker run -it --rm alpine sh` |
| `--entrypoint CMD` | Replace the image's `ENTRYPOINT` for this run. | `--entrypoint ls snip-api -la /bin` |

Full command for my app:
```
docker run --rm --name snip-api -p 8080:8080 --env-file .env.docker snip-api
```

Follow-up: *Does `EXPOSE 8080` in the Dockerfile publish the port?*
Ans: No. `EXPOSE` is documentation: metadata saying "this app listens on 8080". Only `-p` (or `ports:` in compose) actually opens the port to the host.

Follow-up: *Why env vars instead of copying `.env` into the image?*
Ans: The same image must run on my laptop, in CI and in prod with different settings. Config belongs to the environment, not the image (12-factor). Baking `.env` in would also put secrets into every copy of the image and every registry it's pushed to.

---

## 8. What `localhost` means inside a container

**Q: Why can't my containerized app reach Postgres at `localhost:5432`?**
Ans: Every container has its own **network namespace**, so it's effectively its own little computer. `localhost` (`127.0.0.1`) always means "me, this same machine." Inside the container that's the container itself, and Postgres isn't running there. Postgres is on Windows. Docker Desktop gives the host machine the name **`host.docker.internal`**, so the container uses `host.docker.internal:5432`.

**Q: My app listens on `127.0.0.1:8080` and I published `-p 8080:8080`, but requests from Windows fail. Why?**
Ans: Same rule, other direction. `127.0.0.1` means "only accept connections from inside this same machine", i.e. from inside the container. Requests from Windows arrive from **outside** the container, through Docker's port forwarding onto the container's network interface, not its loopback. So the app never sees them. The fix is to listen on **`0.0.0.0:8080`** ("accept connections on every interface"). I set it through the `ADDR` env var, so `127.0.0.1` stays the safe default when running directly on my laptop.

**Q: Then why is `BASE_URL` still `http://localhost:8080`?**
Ans: `BASE_URL` is the address the *user* sees in the short link. The user is on Windows, where `localhost:8080` is the published port, so from their side that's correct.

| Setting | On Windows (`.env`) | Inside the container (`.env.docker`) |
|---|---|---|
| Postgres host | `localhost` | `host.docker.internal` |
| Redis address | `127.0.0.1:6379` | `host.docker.internal:6379` |
| Listen address | `127.0.0.1:8080` | `0.0.0.0:8080` |
| BASE_URL | `http://127.0.0.1:8080` | `http://localhost:8080` |

Follow-up: *What changes with docker-compose (Tuesday)?*
Ans: Compose puts the services on a shared network with built-in DNS, so containers reach each other by **service name**, e.g. `postgres:5432` and `redis:6379`, instead of `host.docker.internal`.

---

## 9. Exec form vs shell form, and non-root

**Q: Why `ENTRYPOINT ["/bin/snip"]` (brackets) and not `ENTRYPOINT /bin/snip`?**
Ans: The bracket (exec) form runs my binary directly as PID 1, so `docker stop` sends SIGTERM straight to it and my graceful-shutdown code runs. The shell form runs `/bin/sh -c /bin/snip`: the shell is PID 1 and doesn't pass the signal on, so Docker waits 10 s and then force-kills with SIGKILL. In-flight requests get cut off.

**Q: Why `adduser` + `USER appuser`?**
Ans: Containers run as root by default. If the app is ever exploited, the attacker gets root inside the container, which makes escaping to the host or tampering much easier. Running as an unprivileged user limits the damage.

---

## Interview summary

- **Images are stacks of content-addressed layers plus a config; containers are running instances with a thin writable layer.** Identical layers are stored once and shared.
- **Tags are mutable pointers to images.** `latest` isn't "newest"; real pipelines tag with the commit SHA.
- **A different image ID doesn't always mean different contents.** BuildKit's provenance attestation changes the manifest-list digest on every build; compare `RootFS.Layers` instead.
- **Cache order: stable first, frequently changing last**, and keep the build context lean with `.dockerignore`.
- **Inside a container, `localhost` is the container itself**: reach the host with `host.docker.internal` (later, other containers by service name), and listen on `0.0.0.0` so published ports work.
