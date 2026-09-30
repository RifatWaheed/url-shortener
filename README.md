# URL Shortener

A URL shortening service written in Go, backed by PostgreSQL.

## Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| [Go](https://go.dev/dl/) | 1.26+ | Must match the `go` line in `go.mod` |
| [PostgreSQL](https://www.postgresql.org/download/) | 13+ recommended | Any recent version that supports `GENERATED ALWAYS AS IDENTITY` (10+) will work |
| [golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate) | v4 | Runs the SQL files in `migrations/` |

Install the migrate CLI with the Postgres driver:

```sh
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

You can also install it with a package manager, for example `scoop install migrate` on Windows or `brew install golang-migrate` on macOS. Check that it installed:

```sh
migrate -version
```

## Configuration (`.env`)

The app loads a `.env` file from the project root when it starts. `.env` is in `.gitignore`, so you need to create your own.

1. Create a database:
   # in psql:  CREATE DATABASE url_shortener;


2. Copy the example file and fill in your credentials:

   ```dotenv
   DATABASE_URL=postgres://postgres:yourpassword@localhost:5432/url_shortener?sslmode=disable
   ADDR=127.0.0.1:8080
   BASE_URL=http://127.0.0.1:8080
   ```

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | yes | — | Postgres connection string, used by both the app and the migrate commands below |
| `ADDR` | no | `127.0.0.1:8080` | Host and port the server listens on. Use `:8080` to listen on all interfaces |
| `BASE_URL` | no | `http://<ADDR>` | Prefix for the `shortUrl` field in responses. Set this to your public domain when the app runs behind a proxy |

If `.env` is missing, the app logs a warning and falls back to the `DATABASE_URL` already set in your environment.

## Migrations

Migrations live in `migrations/` as numbered `up`/`down` pairs:

| # | Creates |
|---|---------|
| 000001 | `users` |
| 000002 | `links` |
| 000003 | `clicks` (plus an index on `link_id`) |
| 000004 | `email_verifications` (plus an index on `user_id`) |
........................
.......................... so on

The migrate CLI doesn't read `.env`, so pass the connection string yourself. Replace the example URL with your own `DATABASE_URL`:

```sh
# Apply all pending migrations
migrate -path migrations -database "postgres://postgres:yourpassword@localhost:5432/url_shortener?sslmode=disable" up

# Roll back the most recent migration
migrate -path migrations -database "<DATABASE_URL>" down 1

# Show the current version
migrate -path migrations -database "<DATABASE_URL>" version
```

To create a new migration pair:

```sh
migrate create -ext sql -dir migrations -seq <name>
```

If a migration fails partway through, the database is marked "dirty". Fix the SQL, then run `migrate ... force <version>` with the last version that applied cleanly, and run `up` again.

## Running the app

```sh
go run .
```

When it starts, the app connects to Postgres, pings it with a 5-second timeout, and listens on `ADDR` (default **`http://127.0.0.1:8080`**). If it can't reach the database, it exits with an error. You should see:

```
connected to database
listening on 127.0.0.1:8080
```

The examples below use the default address.

To build a binary instead:

```sh
go build -o url-shortener .
./url-shortener        # url-shortener.exe on Windows
```

## API

> **Windows users:** in PowerShell, `curl` is an alias for `Invoke-WebRequest`. Use `curl.exe` instead, and note that the JSON quoting below is for bash-style shells. See the PowerShell example after the shorten section.

### `GET /health`

Checks that the server is running.

```sh
curl -i http://127.0.0.1:8080/health
```

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"message":"server is up"}
```

### `POST /shorten`

Creates a short link.

- Body: `{"url": "<http or https URL>"}`
- `Content-Type` must be `application/json`
- The URL must be 2048 characters or fewer and include a host
- Unknown fields are rejected, and the body is capped at 1 MB

```sh
curl -i -X POST http://127.0.0.1:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/some/very/long/path?x=1"}'
```

```http
HTTP/1.1 201 Created
Content-Type: application/json

{"shortCode":"aB3xY9z","shortUrl":"http://127.0.0.1:8080/aB3xY9z"}
```

PowerShell equivalent:

```powershell
curl.exe -i -X POST http://127.0.0.1:8080/shorten -H "Content-Type: application/json" -d '{\"url\": \"https://example.com\"}'
```

Error responses:

| Status | When | Example |
|--------|------|---------|
| `400 Bad Request` | URL missing, malformed, too long, not http/https, or has no host | `-d '{"url": "ftp://example.com"}'` → `url must use http or https` |
| `400 Bad Request` | Invalid JSON, unknown fields, or more than one JSON object | `-d '{"link": "https://example.com"}'` → `invalid JSON body` |
| `415 Unsupported Media Type` | `Content-Type` isn't `application/json` | omit the `-H` header |
| `500 Internal Server Error` | Database failure, or no unique code found after 5 attempts | — |

### `GET /{code}`

Redirects to the original URL.

```sh
# Show the redirect without following it
curl -i http://127.0.0.1:8080/aB3xY9z
```

```http
HTTP/1.1 302 Found
Location: https://example.com/some/very/long/path?x=1
```

```sh
# Follow the redirect
curl -L http://127.0.0.1:8080/aB3xY9z
```

Error responses:

| Status | When |
|--------|------|
| `404 Not Found` | Code doesn't exist, or is invalid (longer than 7 characters or has characters other than `a-z`, `A-Z`, `0-9`) |
| `410 Gone` | The link has an `expires_at` date in the past |
| `500 Internal Server Error` | Database failure |

## Running the tests

The current tests cover short-code generation and input validation. They don't need a database or a running server.

```sh
# Run all tests
go test ./...

# Verbose output, showing each table-driven sub-test
go test -v ./...

# Run a single test
go test -run TestValidateUrl -v ./...

# With coverage
go test -cover ./...
```

| Test | What it checks |
|------|----------------|
| `TestGenerateShortCode` | 10,000 generated codes are the right length, contain only valid characters, and don't collide |
| `TestValidateShortCode` | Empty codes, length limits, and invalid characters |
| `TestValidateUrl` | Empty input, the 2048-character limit, non-http schemes, missing hosts, and valid URLs |

## Project layout

```
.
├── .env.example         # Template for .env
├── main.go              # Loads .env, connects to Postgres, starts the server
├── server.go            # Server struct and route table
├── handlers.go          # HTTP handlers for /health, /shorten, /{code}
├── repository.go        # Database queries (create link, look up link)
├── shortcode.go         # Random 7-character code generator (crypto/rand)
├── validations.go       # URL, short-code, and JSON body validation
├── validations_test.go  # Unit tests
└── migrations/          # golang-migrate SQL files
```
