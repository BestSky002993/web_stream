# video-storage-api

Secure video storage backend.

Built with **Go + Gin + PostgreSQL + pgx + Docker + Air**.
Migrations managed by **golang-migrate**. JWT auth via **golang-jwt/jwt**.

> **Phases shipped so far**
>
> - **Phase 1** — project skeleton, DB pool, graceful shutdown, `/health`.
> - **Phase 2** — real migration system, `admin_users` table, bcrypt
>   password utilities, admin-user repository, `seed-admin` CLI.
> - **Phase 3** — JWT login, lockout policy, protected admin routes.
> - **Phase 4** — `categories` table + admin CRUD with pagination, search,
>   filters, soft delete; protected by the JWT middleware.
> - **Phase 5** — `videos` table + admin upload/CRUD with multipart
>   uploads, private file storage, secure random filenames, MIME sniff
>   validation, per-file size limits, ffmpeg-generated thumbnails,
>   soft delete.
> - **Phase 6** — secure direct streaming with HTTP Range support,
>   HMAC-signed stream URLs, Origin/Referer domain whitelist, public
>   metadata + thumbnail endpoints, structured access logging.
> - **Phase 7** — public video list + random feed with optional per-site
>   daily seed; `thumbnail_url` exposed on every video response.
> - **Phase 8** — `sites` table + admin CRUD, X-API-Key auth middleware,
>   per-site domain whitelist, site-protected list / random / stream-url
>   endpoints (no admin JWT needed for external frontends).
> - **Phase 9** — companion WordPress plugin `video-storage-embed`
>   (sibling directory), shortcode `[video_storage_grid]`, on-demand
>   signed-URL modal player. See [`../video-storage-embed/README.md`](../video-storage-embed/README.md).
> - **Phase 10** — WordPress plugin UI polish: card hover/animations,
>   responsive grid, modal player improvements, new shortcode attributes,
>   accessibility, fallback thumbnails, dark/light theme classes.
> - **Phase 11** — companion `admin-dashboard/` Next.js app (sibling
>   directory), end-to-end web UI for everything Phases 2–8 expose.
>   See [`../admin-dashboard/README.md`](../admin-dashboard/README.md).
> - **Phase 12** — ffprobe-based duration detection at upload time +
>   automatic clamping of `thumbnail_second` against the known
>   duration, with a fallback ladder so ffmpeg failures effectively
>   stop reaching the client.
> - **Phase 13** — HLS pipeline alongside the existing MP4 streaming.
>   Async ffmpeg conversion after upload, status tracked in
>   `videos.hls_*` columns. Signed manifests + segments served under
>   `/api/v1/hls/`, URL issuers for admin + site (`hls-url`),
>   single-variant scaffold designed to host multi-quality later.
> - **Phase 14a** — production Docker setup. Compiled Go binary
>   (no Air), Next.js `next start` (no dev overlay), nginx reverse
>   proxy fronting `/api/` + `/admin/`, Postgres on the internal
>   network only, named volumes for the DB + video/HLS storage.
>   **Run everything from the repo root via the helper scripts in
>   [`../scripts/`](../scripts/)** — see [`../README.md`](../README.md).
> - **Phase 14b** — adaptive multi-quality HLS. ffprobe-driven
>   resolution detection feeds variant selection; the ladder renders
>   360p / 480p / 720p (capped at 720p, never upscales). Aspect ratio
>   and portrait orientation preserved via `scale=-2:H` / `scale=W:-2`.
>   New `videos.hls_variants` column persists the rendered ladder;
>   dashboard surfaces "Qualities: 360p, 480p, 720p" on the edit page.
>   hls.js continues to drive client-side quality switching.
>
> Multi-quality transcoding, queue workers, and analytics come in later phases.

---

## Project structure

```
video-storage-api/
├── cmd/
│   ├── api/main.go              # API entrypoint
│   └── seed-admin/main.go       # one-shot CLI: create the first admin
├── internal/
│   ├── config/                  # env + config loading (incl. AuthConfig)
│   ├── database/                # pgx connection pool
│   ├── routes/                  # gin router wiring
│   ├── middleware/              # logger, request-id, ...
│   ├── health/                  # GET /health handler
│   ├── auth/                    # Phase 3: JWT, login, middleware, /me
│   │   ├── admin_user.go        #   - domain model + role/status consts
│   │   ├── admin_user_repository.go  # - SQL only
│   │   ├── dto.go               #   - request/response shapes
│   │   ├── errors.go            #   - service-level typed errors
│   │   ├── handler.go           #   - HTTP layer (login, me)
│   │   ├── jwt.go               #   - TokenManager (issue / parse)
│   │   ├── middleware.go        #   - RequireAdmin
│   │   ├── module.go            #   - DI wiring + route registration
│   │   ├── password.go          #   - bcrypt utilities
│   │   └── service.go           #   - login + lockout policy
│   ├── sites/                   # Phase 8: external-site API keys + middleware
│   │   ├── site.go              #   - domain model + status consts
│   │   ├── apikey.go            #   - secure random key gen + SHA-256 hash
│   │   ├── errors.go
│   │   ├── dto.go               #   - request/response shapes
│   │   ├── repository.go        #   - SQL only
│   │   ├── service.go           #   - validation + key lifecycle + auth
│   │   ├── handler.go           #   - admin CRUD endpoints
│   │   ├── middleware.go        #   - RequireSite (X-API-Key + domain check)
│   │   └── module.go            #   - DI + route registration
│   ├── videos/                  # Phase 5: upload + admin CRUD
│   │   ├── video.go             #   - domain model + status/visibility consts
│   │   ├── slug.go              #   - local slug normalization
│   │   ├── errors.go
│   │   ├── dto.go               #   - request/response shapes + form bindings
│   │   ├── repository.go        #   - SQL only (joins categories on read)
│   │   ├── service.go           #   - upload pipeline, validation, rollback
│   │   ├── handler.go           #   - multipart HTTP I/O
│   │   └── module.go            #   - DI wiring + route registration
│   ├── categories/              # Phase 4: CRUD module
│   │   ├── category.go          #   - domain model + status consts
│   │   ├── slug.go              #   - testable slug normalization
│   │   ├── slug_test.go
│   │   ├── errors.go            #   - typed service errors
│   │   ├── dto.go               #   - request/response shapes
│   │   ├── repository.go        #   - SQL only
│   │   ├── service.go           #   - validation + business rules
│   │   ├── handler.go           #   - HTTP I/O
│   │   └── module.go            #   - DI wiring + route registration
│   ├── storage/                 # Phase 5: private filesystem backend
│   │   ├── filename.go          #   - SecureFilename + SanitizeStoredFilename
│   │   ├── mime.go              #   - allowed MIMEs + SniffMIME
│   │   ├── local.go             #   - save/delete/open on disk
│   │   └── ffmpeg.go            #   - safe ffmpeg subprocess + frame extraction
│   └── streaming/               # Phase 6: signed-URL streaming + Range
│       ├── signing.go           #   - HMAC-SHA256 signer
│       ├── range.go             #   - RFC 7233 Range parser
│       ├── domain.go            #   - Origin/Referer whitelist
│       ├── mime.go              #   - response Content-Type lookup
│       ├── errors.go            #   - typed service errors
│       ├── service.go           #   - authz + lookup + file open
│       ├── handler.go           #   - HTTP I/O + Range writer + access log
│       └── module.go            #   - DI + route registration
├── migrations/                  # golang-migrate SQL (up/down)
├── docker/
│   └── Dockerfile.dev           # dev image with Air hot reload
├── .env.example
├── .air.toml
├── Dockerfile                   # production multi-stage build
├── docker-compose.yml
├── go.mod
└── README.md
```

---

## Requirements

- Docker 24+
- Docker Compose v2
- (optional, for running outside Docker) Go 1.22+ and `ffmpeg` on `$PATH`

---

## Quick start (Docker)

```bash
# 1. clone & enter
cd video-storage-api

# 2. create your .env from the template, then edit secrets
cp .env.example .env
#    set JWT_SECRET (openssl rand -hex 32) and ADMIN_*

# 3. start the database + api (api hot-reloads via Air)
docker compose up --build

# 4. in a second terminal, apply migrations
docker compose run --rm migrate up

# 5. create the first admin user (idempotent — safe to re-run)
docker compose exec api go run ./cmd/seed-admin
```

The API is then available at <http://localhost:8080>.

---

## Endpoints

| Method | Path                                      | Auth     | Description                                  |
|--------|-------------------------------------------|----------|----------------------------------------------|
| GET    | `/health`                                 | —        | Liveness + DB connectivity                   |
| GET    | `/api/v1`                                 | —        | API root — service metadata                  |
| POST   | `/api/v1/admin/auth/login`                | —        | Email/password → JWT                         |
| GET    | `/api/v1/admin/me`                        | Bearer   | Returns the authenticated admin              |
| POST   | `/api/v1/admin/categories`                | Bearer   | Create a category                            |
| GET    | `/api/v1/admin/categories`                | Bearer   | List categories (paginated, filterable)      |
| GET    | `/api/v1/admin/categories/:id`            | Bearer   | Get one category                             |
| PUT    | `/api/v1/admin/categories/:id`            | Bearer   | Partial update                               |
| DELETE | `/api/v1/admin/categories/:id`            | Bearer   | Soft delete                                  |
| POST   | `/api/v1/admin/videos`                    | Bearer   | Upload a video (multipart/form-data)         |
| GET    | `/api/v1/admin/videos`                    | Bearer   | List videos (paginated, filterable)          |
| GET    | `/api/v1/admin/videos/:id`                | Bearer   | Get one video                                |
| PUT    | `/api/v1/admin/videos/:id`                | Bearer   | Partial update (optional thumbnail swap)     |
| DELETE | `/api/v1/admin/videos/:id`                | Bearer   | Soft delete (status→deleted)                 |
| POST   | `/api/v1/admin/videos/:id/stream-url`     | Bearer   | Issue a temporary signed stream URL          |
| GET    | `/api/v1/videos`                          | —        | Public list (paginated, filterable, sortable)|
| GET    | `/api/v1/videos/random`                   | —        | Public random feed (optionally seeded)       |
| GET    | `/api/v1/videos/:slug`                    | —        | Public metadata (only `public` + `ready`)    |
| GET    | `/api/v1/stream/:videoID`                 | Signed/Public | Range-aware video stream                |
| GET    | `/api/v1/thumbnails/:videoID`             | —        | Auto-generated thumbnail (jpg)               |
| POST   | `/api/v1/admin/sites`                     | Bearer   | Create site + return plain API key (once)    |
| GET    | `/api/v1/admin/sites`                     | Bearer   | List sites (no key material)                 |
| GET    | `/api/v1/admin/sites/:id`                 | Bearer   | Get site                                     |
| PUT    | `/api/v1/admin/sites/:id`                 | Bearer   | Update site                                  |
| POST   | `/api/v1/admin/sites/:id/disable`         | Bearer   | Set `status=disabled`                        |
| POST   | `/api/v1/admin/sites/:id/rotate-key`      | Bearer   | New API key (old one stops working)          |
| DELETE | `/api/v1/admin/sites/:id`                 | Bearer   | Soft delete                                  |
| GET    | `/api/v1/admin/sites/:id/publications`    | Bearer   | List publication rows for a site             |
| GET    | `/api/v1/admin/videos/:id/publications`   | Bearer   | List publication rows for a video            |
| PUT    | `/api/v1/admin/sites/:id/videos/:video_id/publication` | Bearer | Upsert a site/video publication row |
| POST   | `/api/v1/admin/publications/:id/approve`  | Bearer   | Approve publication for site sync            |
| POST   | `/api/v1/admin/publications/:id/disable`  | Bearer   | Hide publication from site sync              |
| POST   | `/api/v1/admin/publications/:id/resync`   | Bearer   | Mark publication for resync                  |
| DELETE | `/api/v1/admin/publications/:id`          | Bearer   | Remove publication row                       |
| GET    | `/api/v1/admin/sites/:id/remote-categories` | Bearer | List imported WP categories for a site |
| GET    | `/api/v1/admin/sites/:id/category-mappings` | Bearer | List per-site backend-to-WP category mappings |
| PUT    | `/api/v1/admin/sites/:id/category-mappings` | Bearer | Save stream-side category mapping contract |
| POST   | `/api/v1/admin/sites/:id/category-mappings/suggest` | Bearer | Create disabled same-name mapping suggestions |
| GET    | `/api/v1/site/videos`                     | X-API-Key | Public list (site-authenticated)            |
| GET    | `/api/v1/site/videos/random`              | X-API-Key | Random feed (defaults seed to site ID)      |
| GET    | `/api/v1/site/sync/videos`                | X-API-Key | Approved/synced videos for controlled WP sync |
| GET    | `/api/v1/site/sync/categories`            | X-API-Key | API categories for WP taxonomy mapping      |
| POST   | `/api/v1/site/sync/categories/import`   | X-API-Key | Import local WP categories for this site    |
| GET    | `/api/v1/site/sync/category-mappings`    | X-API-Key | List mapping contract scoped to this site   |
| PUT    | `/api/v1/site/sync/category-mappings/:id` | X-API-Key | Confirm/update WP side of a mapping         |
| POST   | `/api/v1/site/sync/videos/:id/status`     | X-API-Key | Record remote WP sync result                |
| POST   | `/api/v1/site/videos/:id/stream-url`      | X-API-Key | Signed URL for a public+ready video         |

---

## Authentication

### Login

```bash
curl -X POST http://localhost:8080/api/v1/admin/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email":"admin@example.com",
    "password":"your_password"
  }'
```

Successful response:

```json
{
  "token": "eyJhbGciOi...",
  "expires_in": 86400,
  "admin": {
    "id": "9b1f...",
    "email": "admin@example.com",
    "display_name": "Site Administrator",
    "role": "super_admin"
  }
}
```

Failure responses (all returned as `HTTP 401` with `{"error": "..."}`):

| Condition                                    | Body                                  |
|----------------------------------------------|---------------------------------------|
| Unknown email **or** wrong password           | `{"error":"invalid credentials"}`     |
| Within a lockout window                       | `{"error":"account is temporarily locked"}` |
| `status = 'disabled'`                          | `{"error":"account is disabled"}`     |
| Malformed JSON / missing email / bad format   | `400` `{"error":"invalid request body"}` |

### Calling a protected route

```bash
TOKEN="paste-token-here"

curl http://localhost:8080/api/v1/admin/me \
  -H "Authorization: Bearer $TOKEN"
```

Response:

```json
{
  "id": "9b1f...",
  "email": "admin@example.com",
  "display_name": "Site Administrator",
  "role": "super_admin",
  "status": "active"
}
```

The middleware rejects (HTTP 401) any request where the header is missing,
the token is malformed/expired/badly-signed, the referenced admin no longer
exists, or `status` is not `active`.

### Lockout policy

- Each wrong password increments `admin_users.failed_login_attempts`.
- When the counter reaches `LOGIN_MAX_ATTEMPTS` (default `5`), the account
  is set to `status = 'locked'` and `locked_until = now() + LOGIN_LOCK_DURATION_MINUTES`
  (default 15 minutes) in a single atomic UPDATE.
- While `locked_until` is in the future, login attempts are rejected
  *before* the password is even checked — so brute-forcers can't keep
  pushing the timer.
- A successful login (which can only happen after the lock expires)
  resets `failed_login_attempts = 0`, clears `locked_until`, stamps
  `last_login_at`, and flips `status` back to `active`.

### Environment knobs

| Variable                       | Default | Notes                                   |
|--------------------------------|---------|-----------------------------------------|
| `JWT_SECRET`                   | (req.)  | ≥32 chars outside `APP_ENV=development` |
| `JWT_EXPIRES_IN`               | `24h`   | any `time.ParseDuration` value          |
| `LOGIN_MAX_ATTEMPTS`           | `5`     | wrong-password threshold                |
| `LOGIN_LOCK_DURATION_MINUTES`  | `15`    | lockout window                          |

Generate a JWT secret:

```bash
openssl rand -hex 32
```

---

## CORS (browser callers)

Browser clients (the Next.js admin dashboard, custom React frontends,
test pages) trigger a preflight `OPTIONS` before any request with a
non-simple method or an `Authorization` header. Without CORS the
preflight returns `404` from `NoRoute` and the real request never
fires — symptom: dashboard login fails with no obvious error.

The API ships a global CORS middleware (registered before route
matching, after `gin.Recovery`):

- exact-match allow-list driven by `ALLOWED_CORS_ORIGINS`
- echoes the request `Origin` into `Access-Control-Allow-Origin`
  (never `*`, so it's safe with `Authorization`)
- answers `OPTIONS` with `204 No Content` before the router runs
- emits `Vary: Origin` on every response
- allows: `Authorization`, `Content-Type`, `X-API-Key`, `X-Request-ID`,
  `Accept`, `Range`
- exposes: `X-Request-ID`, `Content-Range`, `Content-Length`,
  `Accept-Ranges`
- `Access-Control-Allow-Credentials: true` is on by default so the
  dashboard's planned cookie-based auth phase will work too

### Configure

```bash
# .env
ALLOWED_CORS_ORIGINS=http://localhost:3000,http://127.0.0.1:3000,http://testplugin.local
```

Empty (the default) means no cross-origin browser access — useful for
production deployments where the dashboard is served from the same
origin as the API. Add the dashboard's exact origin
(scheme + host + port) for development.

### Verify the preflight

```bash
curl -i -X OPTIONS http://localhost:8080/api/v1/admin/auth/login \
  -H 'Origin: http://localhost:3000' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: Authorization, Content-Type'

# Expected:
#   HTTP/1.1 204 No Content
#   Access-Control-Allow-Origin: http://localhost:3000
#   Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD
#   Access-Control-Allow-Headers: Authorization, Content-Type, X-API-Key, …
#   Access-Control-Allow-Credentials: true
#   Vary: Origin
```

If the response is `404 route not found`, the middleware didn't run —
double-check the env var spelling and restart the container after
editing `.env`.

If the response is `204` but the `Access-Control-Allow-Origin` header
is missing, the request Origin isn't in your allow-list. Add it.

---

## Categories (admin)

All category endpoints live under `/api/v1/admin/categories` and require
a valid admin JWT (`Authorization: Bearer <token>`). Missing, expired, or
otherwise-invalid tokens return `401`; non-`active` admin accounts are
rejected even with a valid signature.

### Create

```bash
curl -X POST http://localhost:8080/api/v1/admin/categories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tutorials","slug":"tutorials","description":"Tutorial videos","status":"active","sort_order":1}'
```

Returns `201 Created` with:

```json
{
  "id": "9b1f...",
  "name": "Tutorials",
  "slug": "tutorials",
  "description": "Tutorial videos",
  "status": "active",
  "sort_order": 1,
  "created_at": "2026-05-23T...",
  "updated_at": "2026-05-23T..."
}
```

Slugs are normalized server-side: trimmed, lower-cased, non-alphanumeric
characters become `-`, runs collapse, leading/trailing `-` are stripped.
A slug whose normalized form is empty returns `400 invalid slug`.
A slug colliding with an existing live category returns `409 Conflict`.

### List

```bash
curl -G http://localhost:8080/api/v1/admin/categories \
  -H "Authorization: Bearer $TOKEN" \
  --data-urlencode 'page=1' \
  --data-urlencode 'limit=20' \
  --data-urlencode 'search=tutorial' \
  --data-urlencode 'status=active' \
  --data-urlencode 'sort=sort_order'
```

| Query param | Default | Notes                                                       |
|-------------|---------|-------------------------------------------------------------|
| `page`      | `1`     | 1-indexed                                                   |
| `limit`     | `20`    | clamped to `[1, 100]`                                       |
| `search`    | —       | ILIKE match on `name` and `slug`                            |
| `status`    | —       | `active` or `disabled`                                      |
| `sort`      | —       | whitelisted: `sort_order`, `-sort_order`, `name`, `-name`, `created_at`, `-created_at` (default: `sort_order` asc, then `created_at` desc) |

Response:

```json
{
  "data": [
    {"id":"...", "name":"Tutorials", "slug":"tutorials", ...}
  ],
  "pagination": {"page":1, "limit":20, "total":1, "total_pages":1}
}
```

Soft-deleted categories are excluded.

### Get one

```bash
curl http://localhost:8080/api/v1/admin/categories/$CATEGORY_ID \
  -H "Authorization: Bearer $TOKEN"
```

Returns `404` if not found or soft-deleted.

### Update (partial)

```bash
curl -X PUT http://localhost:8080/api/v1/admin/categories/$CATEGORY_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tutorials & Guides","sort_order":2}'
```

All fields are optional. A field that is omitted (or sent as `null`) is
left unchanged. To clear a `description`, send `"description": ""`.
Slug uniqueness is re-checked only when `slug` is included — a
collision returns `409`. Updating a soft-deleted row returns `404`.

### Soft delete

```bash
curl -X DELETE http://localhost:8080/api/v1/admin/categories/$CATEGORY_ID \
  -H "Authorization: Bearer $TOKEN"
```

Returns `204 No Content`. The row is kept in the DB with
`deleted_at = NOW()` and disappears from the standard list. A second
delete of the same id returns `404`.

### Schema: `categories`

| Column        | Type         | Notes                                                |
|---------------|--------------|------------------------------------------------------|
| `id`          | UUID         | PK, defaults to `gen_random_uuid()`                  |
| `name`        | TEXT         | required, non-blank                                  |
| `slug`        | TEXT         | required; unique among live rows (case-insensitive)  |
| `description` | TEXT         | nullable                                             |
| `status`      | TEXT         | CHECK: `active` / `disabled`; defaults to `active`   |
| `sort_order`  | INTEGER      | defaults to `0`                                      |
| `created_at`  | TIMESTAMPTZ  | defaults to `NOW()`                                  |
| `updated_at`  | TIMESTAMPTZ  | bumped on every update                               |
| `deleted_at`  | TIMESTAMPTZ  | nullable — soft delete                               |

### End-to-end smoke test

```bash
# 1. apply Phase 4 migration
docker compose run --rm migrate up

# 2. log in to capture a token (assumes the seed-admin user exists)
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/admin/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"change-me-please"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')

# 3. create
CAT_ID=$(curl -s -X POST http://localhost:8080/api/v1/admin/categories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tutorials","slug":"Tutorials!","description":"Tutorial videos","sort_order":1}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')

# 4. duplicate slug should 409
curl -i -X POST http://localhost:8080/api/v1/admin/categories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Dupes","slug":"tutorials"}'

# 5. list
curl -s http://localhost:8080/api/v1/admin/categories \
  -H "Authorization: Bearer $TOKEN"

# 6. update
curl -s -X PUT http://localhost:8080/api/v1/admin/categories/$CAT_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"sort_order":5,"status":"disabled"}'

# 7. soft delete (204)
curl -i -X DELETE http://localhost:8080/api/v1/admin/categories/$CAT_ID \
  -H "Authorization: Bearer $TOKEN"

# 8. list again — deleted row is gone
curl -s http://localhost:8080/api/v1/admin/categories \
  -H "Authorization: Bearer $TOKEN"
```

---

## Videos (admin)

All video endpoints live under `/api/v1/admin/videos` and require a
valid admin JWT. Uploaded files are stored **outside any public web
path** (default `storage/private/{videos,thumbnails}`) under random
server-generated filenames; the client-supplied filename is preserved
only as metadata.

**Thumbnails are not uploaded** — the API runs `ffmpeg` against the
uploaded video to extract a frame at the requested `thumbnail_second`.
ffmpeg is installed inside both the dev and production Docker images;
running outside Docker requires it on `$PATH`.

### File-handling rules

- The HTTP body is wrapped in `http.MaxBytesReader` so an oversized
  request is rejected as it arrives. Cap = `MAX_VIDEO_UPLOAD_SIZE_MB`
  + 1 MiB headroom for the multipart envelope and text fields.
- After parsing, the video part is also size-checked against
  `MAX_VIDEO_UPLOAD_SIZE_MB`.
- The first 512 bytes of the video are sniffed via
  `http.DetectContentType` and matched against the allowlist:
  `video/mp4`, `video/webm`, `video/quicktime` (mp4/webm/mov). The
  client's `Content-Type` is not trusted.
- The video is streamed via `io.Copy` from the multipart reader to
  disk — Go's `mime/multipart` parser already spills large parts to a
  temp file, so the API never holds an entire video in memory.
- After the video is saved, the API runs:
  `ffmpeg -loglevel error -y -ss <second> -i <video> -frames:v 1 -q:v 2 <thumb>.jpg`
  Each arg is a discrete `exec.Command` argument — no shell, no string
  concatenation. The subprocess is capped at a 60s timeout. If ffmpeg
  exits non-zero (e.g. seek past EOF or corrupt input), the saved
  video file is deleted and the request returns `422`.
- Disk filename = 32 hex chars of `crypto/rand` + canonical extension.
  The video file is opened with `O_EXCL`; the thumbnail is written by
  ffmpeg with `-y` and verified non-empty afterward.
- Title, slug, description, etc. are validated separately. Slug is
  normalized just like categories (lower-case, kebab-case, ASCII-only).

### Upload (Create)

```bash
curl -X POST http://localhost:8080/api/v1/admin/videos \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=My First Video" \
  -F "slug=my-first-video" \
  -F "description=Test upload" \
  -F "category_id=$CATEGORY_ID" \
  -F "visibility=private" \
  -F "thumbnail_second=5" \
  -F "video=@./video.mp4"
```

Required parts: `title`, `slug`, `category_id`, `video`.
Optional parts: `description`, `visibility` (default `private`),
`thumbnail_second` (default `1`).

There is **no** `thumbnail` file part — the API extracts the frame at
`thumbnail_second` from the uploaded video.

Returns `201 Created` with:

```json
{
  "id": "uuid",
  "title": "My First Video",
  "slug": "my-first-video",
  "description": "Test upload",
  "status": "ready",
  "visibility": "private",
  "category": {"id": "...", "name": "Tutorials"},
  "video": {
    "filename": "ab12...ef.mp4",
    "original_name": "video.mp4",
    "mime_type": "video/mp4",
    "size_bytes": 12345678
  },
  "thumbnail": {"filename": "cd34...ef.jpg", "second": 5},
  "duration_seconds": null,
  "views_count": 0,
  "uploaded_by_admin_id": "uuid",
  "created_at": "...",
  "updated_at": "..."
}
```

Failure responses:

| Condition                                          | Status | Body                                  |
|----------------------------------------------------|--------|---------------------------------------|
| Missing/invalid token                              | `401`  | `{"error":"unauthorized"}`            |
| Missing required field (title, slug, category_id…) | `400`  | `{"error":"invalid request: ..."}`    |
| `category_id` not found / soft-deleted             | `400`  | `{"error":"category not found"}`      |
| Slug normalizes to empty                           | `400`  | `{"error":"invalid slug"}`            |
| Duplicate slug                                     | `409`  | `{"error":"video slug already exists"}` |
| Disallowed video MIME                              | `400`  | `{"error":"unsupported video format"}` |
| `thumbnail_second` not an integer / out of range   | `400`  | `{"error":"invalid thumbnail_second"}` |
| ffmpeg couldn't extract a frame (e.g. past EOF)    | `422`  | `{"error":"could not generate thumbnail at the requested second"}` |
| Body / per-file size cap exceeded                  | `413`  | `{"error":"video file exceeds size limit"}` |

### List

```bash
curl -G http://localhost:8080/api/v1/admin/videos \
  -H "Authorization: Bearer $TOKEN" \
  --data-urlencode 'page=1' \
  --data-urlencode 'limit=20' \
  --data-urlencode 'search=intro' \
  --data-urlencode 'category_id=$CATEGORY_ID' \
  --data-urlencode 'status=ready' \
  --data-urlencode 'visibility=private' \
  --data-urlencode 'sort=-created_at'
```

| Query param   | Default | Notes                                                  |
|---------------|---------|--------------------------------------------------------|
| `page`        | `1`     | 1-indexed                                              |
| `limit`       | `20`    | clamped to `[1, 100]`                                  |
| `search`      | —       | ILIKE on `title` and `slug`                            |
| `category_id` | —       | UUID                                                   |
| `status`      | —       | `draft` / `processing` / `ready` / `blocked` / `deleted` |
| `visibility`  | —       | `public` / `private`                                   |
| `sort`        | `-created_at` | `title`, `-title`, `created_at`, `-created_at`, `views`, `-views` |

Soft-deleted rows are always excluded.

### Get one

```bash
curl http://localhost:8080/api/v1/admin/videos/$VIDEO_ID \
  -H "Authorization: Bearer $TOKEN"
```

Returns `404` if not found or soft-deleted.

### Update (partial, optional thumbnail regeneration)

```bash
# text fields only
curl -X PUT http://localhost:8080/api/v1/admin/videos/$VIDEO_ID \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=My Updated Title" \
  -F "status=blocked"

# regenerate the thumbnail from the existing video at second 12
curl -X PUT http://localhost:8080/api/v1/admin/videos/$VIDEO_ID \
  -H "Authorization: Bearer $TOKEN" \
  -F "thumbnail_second=12"
```

Multipart form. Empty/omitted text fields mean "no change". If
`thumbnail_second` is included, the API re-runs ffmpeg against the
stored video at that second, writes a new thumbnail file, and updates
the DB. The old thumbnail is deleted only after the UPDATE commits
successfully. A failed DB UPDATE deletes the new file so nothing is
orphaned. Status cannot be set to `deleted` via update — use the
DELETE endpoint.

### Soft delete

```bash
curl -X DELETE http://localhost:8080/api/v1/admin/videos/$VIDEO_ID \
  -H "Authorization: Bearer $TOKEN"
```

Returns `204 No Content`. Sets `deleted_at = NOW()` and `status = 'deleted'`.
The stored video and thumbnail files are intentionally **not** removed
in Phase 5 — a background file-cleanup job will reclaim them in a later
phase.

### Storage / processing knobs

| Variable                       | Default                            | Notes                                |
|--------------------------------|------------------------------------|--------------------------------------|
| `VIDEO_STORAGE_PATH`           | `storage/private/videos`           | private dir; created on startup      |
| `THUMBNAIL_STORAGE_PATH`       | `storage/private/thumbnails`       | private dir; created on startup      |
| `MAX_VIDEO_UPLOAD_SIZE_MB`     | `2048`                             | per-video upload cap                 |
| `MAX_THUMBNAIL_UPLOAD_SIZE_MB` | `10`                               | reserved — thumbnails are no longer uploaded |

`ffmpeg` must be present on `$PATH`. Both `docker/Dockerfile.dev` and
the production `Dockerfile` install it via `apk add --no-cache ffmpeg`.

The Docker dev setup adds a named volume (`storage_data` → `/app/storage`)
so uploaded files survive container rebuilds. The bind-mounted source
tree (`./storage` on the host) also picks them up, so files are visible
locally for inspection.

### Schema: `videos`

| Column                  | Type         | Notes                                                |
|-------------------------|--------------|------------------------------------------------------|
| `id`                    | UUID         | PK, defaults to `gen_random_uuid()`                  |
| `category_id`           | UUID         | FK → `categories(id)`; non-deleted enforced in service |
| `title`                 | TEXT         | required, non-blank                                  |
| `slug`                  | TEXT         | required; unique among live rows (case-insensitive)  |
| `description`           | TEXT         | nullable                                             |
| `status`                | TEXT         | CHECK: `draft`/`processing`/`ready`/`blocked`/`deleted`; defaults to `draft` |
| `visibility`            | TEXT         | CHECK: `public`/`private`; defaults to `private`     |
| `video_filename`        | TEXT         | server-generated random name                         |
| `video_original_name`   | TEXT         | sanitized client-supplied name (display only)        |
| `video_mime_type`       | TEXT         | sniffed MIME                                         |
| `video_size_bytes`      | BIGINT       | actual bytes written                                 |
| `thumbnail_filename`    | TEXT         | server-generated random name (ffmpeg output)         |
| `thumbnail_second`      | INTEGER      | timestamp the frame was captured at (defaults to 1)  |
| `duration_seconds`      | INTEGER      | nullable — populated by future processing pipeline   |
| `views_count`           | BIGINT       | defaults to 0                                        |
| `uploaded_by_admin_id`  | UUID         | FK → `admin_users(id)`; populated from JWT subject   |
| `created_at`            | TIMESTAMPTZ  | defaults to `NOW()`                                  |
| `updated_at`            | TIMESTAMPTZ  | bumped on every update                               |
| `deleted_at`            | TIMESTAMPTZ  | nullable — soft delete                               |

### End-to-end smoke test

```bash
# 1. apply migrations (Phase 5 + thumbnail_second addition)
docker compose run --rm migrate up

# 2. log in to get a token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/admin/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"change-me-please"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')

# 3. create a category (or reuse an existing one)
CATEGORY_ID=$(curl -s -X POST http://localhost:8080/api/v1/admin/categories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Tutorials","slug":"tutorials"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')

# 4. upload — thumbnail is generated by ffmpeg at second 5
VIDEO_ID=$(curl -s -X POST http://localhost:8080/api/v1/admin/videos \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=My First Video" -F "slug=my-first-video" \
  -F "category_id=$CATEGORY_ID" -F "visibility=private" \
  -F "thumbnail_second=5" \
  -F "video=@./video.mp4" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["id"])')

# 5. list
curl -s http://localhost:8080/api/v1/admin/videos \
  -H "Authorization: Bearer $TOKEN"

# 6. update text fields
curl -s -X PUT http://localhost:8080/api/v1/admin/videos/$VIDEO_ID \
  -H "Authorization: Bearer $TOKEN" \
  -F "status=blocked"

# 7. regenerate thumbnail at second 12
curl -s -X PUT http://localhost:8080/api/v1/admin/videos/$VIDEO_ID \
  -H "Authorization: Bearer $TOKEN" \
  -F "thumbnail_second=12"

# 8. soft delete
curl -i -X DELETE http://localhost:8080/api/v1/admin/videos/$VIDEO_ID \
  -H "Authorization: Bearer $TOKEN"

# 9. files remain on disk (Phase 5 keeps them; cleanup worker comes later)
ls storage/private/videos
ls storage/private/thumbnails
```

### Duration detection + thumbnail clamping (Phase 12)

Uploads now run `ffprobe` once after the video lands on disk, and
store the integer-seconds duration in `videos.duration_seconds`.
That duration is then used by every thumbnail call to clamp
`thumbnail_second` to a safe value.

**Probe command** (run only once per upload, never per request):

```
ffprobe -v error -show_entries format=duration -of json <input>
```

Sub-second videos round up to `1`; probe failures store `NULL` and
keep going — the admin can still upload broken or exotic containers.

**Clamping rule** (`internal/storage/ffmpeg.go: ClampThumbnailSecond`):

| Requested second | Duration known? | Used second                  |
|------------------|-----------------|------------------------------|
| `< 0`            | —               | `0`                          |
| `>= duration`    | yes             | `duration - 1`               |
| `>= duration`    | no (probe failed) | passed through (fallback ladder catches failures) |
| any              | yes, `>= 1`     | the requested value          |

Example: a 38-second video with `thumbnail_second=68` results in a
thumbnail at second 37, and the response body shows
`thumbnail.second = 37`. The dashboard refreshes to the corrected
value.

**Fallback ladder** for unknown-duration uploads (or genuinely corrupt
files): if ffmpeg fails at the clamped second, the storage layer
silently retries at second `1`, then at second `0`. Only if all three
fail does the API surface
`422 thumbnail generation failed; please try a lower thumbnail second
or re-upload the video`.

**Edit path** (`PUT /api/v1/admin/videos/:id` with
`thumbnail_second=...`): the service reuses the stored `duration_seconds`
to clamp without re-probing. Videos uploaded *before* Phase 12 don't
have a duration yet — those get a lazy probe on first edit, with the
result persisted alongside the thumbnail update.

### Troubleshooting thumbnail generation

| Symptom                                                  | Likely cause                                                  |
|----------------------------------------------------------|---------------------------------------------------------------|
| 422 "thumbnail generation failed" after upload           | Video file unreadable by ffmpeg. Re-encode and try again.     |
| `thumbnail.second` in response differs from your request | Auto-clamp fired. Compare your value against `duration_seconds`. |
| `duration_seconds` is `null`                             | ffprobe failed or returned no duration. Check the API container logs for `storage: ffprobe failed for …`. |
| Update accepts `thumbnail_second` but returns the same value | Backend clamped to the same value (e.g. you sent `0` on a 1-second video). |

---

## Streaming (Phase 6)

The private storage directory is **never** served directly. All video
bytes flow through the API:

| Method | Path                                    | Notes                                    |
|--------|-----------------------------------------|------------------------------------------|
| GET    | `/api/v1/stream/:videoID`               | Range-aware bytes; signed URL required   |
| HEAD   | `/api/v1/stream/:videoID`               | Same headers as GET, no body             |
| GET    | `/api/v1/videos/:slug`                  | Public metadata (only `public` + `ready`)|
| GET    | `/api/v1/thumbnails/:videoID`           | Auto-generated thumbnail (always served) |
| POST   | `/api/v1/admin/videos/:id/stream-url`   | Admin-only: returns a signed URL         |

### Security stack

Each public request is filtered through, in order:

1. **Domain whitelist (anti-hotlink)** — if `Origin` is present it must
   match `ALLOWED_STREAM_DOMAINS`; if Origin is absent and `Referer` is
   present, Referer must match. Requests with neither header (curl,
   server-to-server) are allowed — they're still gated by the signed
   URL requirement. Matching is case-insensitive and matches
   subdomain suffixes: `example.com` allows `cdn.example.com`. An
   empty list or `*` disables enforcement (dev default).
2. **Lookup** — the video must exist and not be soft-deleted, else 404.
3. **Status** — only `status='ready'` is streamable, else 404
   (so admins can't accidentally publish drafts).
4. **Signed URL** — a valid `?expires=…&signature=…` is required for
   every MP4 byte response. Visibility controls metadata/listing and
   whether a site API key may issue a signed URL.
5. **Filename sanitisation** — the on-disk filename comes from the DB
   and is re-validated via `storage.SanitizeStoredFilename` before any
   filesystem call. Client URL parts never reach `filepath.Join`.

### Signed URL format

```
<PUBLIC_BASE_URL>/api/v1/stream/<videoID>?expires=<unix>&signature=<hex>
```

- `expires` is a unix timestamp in seconds.
- `signature` is the lowercase hex HMAC-SHA256 of `stream:<videoID>:<expires>`,
  keyed with `STREAM_SIGNING_SECRET`.
- Verification uses constant-time comparison (`hmac.Equal`).

```bash
# Admin: get a signed URL (good for STREAM_URL_EXPIRATION_MINUTES)
TOKEN=...   # admin JWT
VIDEO_ID=...
curl -s -X POST \
    http://localhost:8080/api/v1/admin/videos/$VIDEO_ID/stream-url \
    -H "Authorization: Bearer $TOKEN"
```

Response:

```json
{
  "stream_url": "http://localhost:8080/api/v1/stream/<id>?expires=1716480000&signature=abc...",
  "expires_at": "2026-05-23T15:00:00Z",
  "expires": 1716480000
}
```

### Streaming with HTTP Range

```bash
# Full file (200)
curl -i "$STREAM_URL"

# First megabyte (206)
curl -i -H "Range: bytes=0-1048575" "$STREAM_URL"

# Tail of the file (206) — last 64 KiB
curl -i -H "Range: bytes=-65536" "$STREAM_URL"

# Open-ended (206) — from 5 MB to EOF
curl -i -H "Range: bytes=5242880-" "$STREAM_URL"

# HEAD — headers only
curl -I "$STREAM_URL"
```

Response headers:

| Header           | When sent | Example                                |
|------------------|-----------|----------------------------------------|
| `Accept-Ranges`  | always    | `bytes`                                |
| `Content-Type`   | always    | `video/mp4`                            |
| `Content-Length` | always    | bytes of the response body             |
| `Content-Range`  | 206 only  | `bytes 0-1048575/12345678`             |
| `Cache-Control`  | always    | `private, max-age=300`                 |

Status codes:

| Condition                          | Status |
|------------------------------------|--------|
| Full body                          | `200`  |
| Successful range                   | `206`  |
| Malformed / unsatisfiable `Range`  | `416` (with `Content-Range: bytes */<size>`) |
| Missing signature on private video | `401`  |
| Expired or bad signature           | `401`  |
| Origin/Referer not on whitelist    | `403`  |
| Unknown / soft-deleted / not ready | `404`  |

### Public metadata

```bash
curl http://localhost:8080/api/v1/videos/my-first-video
```

```json
{
  "id": "...",
  "title": "My First Video",
  "slug": "my-first-video",
  "description": "Test upload",
  "thumbnail_url": "http://localhost:8080/api/v1/thumbnails/<id>",
  "duration_seconds": null,
  "visibility": "public",
  "available": true
}
```

Private videos return `404` — they can only be discovered through the
signed URL the admin issues directly.

### Thumbnails

```bash
curl -i http://localhost:8080/api/v1/thumbnails/$VIDEO_ID -o /tmp/t.jpg
```

Always served (subject to the same domain whitelist). `Content-Type` is
`image/jpeg`; `Cache-Control: public, max-age=300`.

### Access log

Every request to `/api/v1/stream/:id` emits a single line prefixed
`[STREAM]` to stdout:

```
[STREAM] video_id=… ip=… ua="…" status=206 bytes=1048576 range="bytes=0-1048575" took=42ms
```

Passwords and signatures are never logged. Database persistence is
deferred to a later analytics phase.

### Streaming config

| Variable                        | Default     | Notes                                   |
|---------------------------------|-------------|-----------------------------------------|
| `STREAM_SIGNING_SECRET`         | (required)  | HMAC-SHA256 key; ≥32 chars outside `APP_ENV=development` |
| `STREAM_URL_EXPIRATION_MINUTES` | `60`        | TTL of admin-issued links               |
| `ALLOWED_STREAM_DOMAINS`        | `""`        | comma-separated; empty / `*` allows all |
| `PUBLIC_BASE_URL`               | `""`        | absolute base for signed URLs; empty → derived from request |

### Local-dev setup

`ALLOWED_STREAM_DOMAINS=localhost` is enough to embed a stream into a
local WordPress instance on `http://localhost:<port>`. For a quick
browser test, open the signed URL directly — browsers don't send Origin
for top-level GETs, so the whitelist check falls through and only the
signature gates access.

### End-to-end smoke test

```bash
# 1. log in + create a public video (see Phase 5 smoke test)
TOKEN=...     # admin JWT
VIDEO_ID=...  # status=ready, visibility=private

# 2. issue a signed URL
SIGNED=$(curl -s -X POST \
    http://localhost:8080/api/v1/admin/videos/$VIDEO_ID/stream-url \
    -H "Authorization: Bearer $TOKEN" \
    | python3 -c 'import sys,json;print(json.load(sys.stdin)["stream_url"])')

# 3. tail of the file with a Range request (expect 206)
curl -i -H "Range: bytes=-65536" "$SIGNED" -o /tmp/tail.bin
file /tmp/tail.bin

# 4. tampered signature → 401
curl -i "${SIGNED%?}X"

# 5. hotlink from a disallowed origin → 403 (if whitelist is configured)
curl -i -H "Origin: https://evil.example.com" "$SIGNED"

# 6. fetch public metadata (a public video)
curl http://localhost:8080/api/v1/videos/<public-slug>

# 7. fetch thumbnail
curl -o /tmp/t.jpg http://localhost:8080/api/v1/thumbnails/$VIDEO_ID
```

---

## HLS streaming (Phase 13)

HLS runs **alongside** the MP4 streaming layer — both are signed with
the same `STREAM_SIGNING_SECRET` (under different signing scopes), and
both can serve playback for the same video. The admin / site picks
whichever URL they want; the existing `/api/v1/stream/...` endpoints
keep working unchanged.

### Pipeline

```
upload   →  insert row (hls_status='pending')
         →  hls.Service.Start  (background goroutine)
         →  hls_status='processing'
         →  ffmpeg -c copy -f hls ...
         →  write storage/private/hls/<id>/v1/index.m3u8 + segment*.ts
         →  write storage/private/hls/<id>/master.m3u8
         →  hls_status='ready', hls_ready_at=NOW()
```

If anything fails, `hls_status` flips to `failed` and `hls_error` holds
a 500-char-truncated copy of the error. The dashboard surfaces this on
the video edit page. MP4 streaming keeps working regardless.

### Storage layout (Phase 14: multi-quality)

```
storage/private/hls/
└── <video-id>/
    ├── master.m3u8           ← references every rendered variant
    ├── 360p/
    │   ├── index.m3u8
    │   ├── segment000.ts
    │   └── …
    ├── 480p/
    │   └── …
    └── 720p/
        └── …
```

### Supported qualities

| Variant | Short side | Video bitrate | Audio bitrate | Master BANDWIDTH |
|---------|------------|---------------|---------------|------------------|
| 360p    | 360 px     | 800 k         | 96 k          | 900 000          |
| 480p    | 480 px     | 1400 k        | 128 k         | 1 530 000        |
| 720p    | 720 px     | 2500 k        | 128 k         | 2 630 000        |

**Why 720p is the ceiling.** The platform targets web embeds + WordPress
front-ends, where 720p is the practical sweet spot: 1080p quadruples the
encode time + storage for marginal perceived quality in a sub-1080p
viewport. Future phases can add 1080p (and beyond) — the rung is a
single entry on `DefaultVariants` away.

### Variant selection — never upscale

ffprobe detects the source's width × height before any encode runs.
`SelectVariants` then keeps only the rungs whose `TargetShortSide` is
≤ `min(srcWidth, srcHeight)`:

| Source         | Rendered variants     |
|----------------|-----------------------|
| 1920×1080      | 360p, 480p, 720p      |
| 1280×720       | 360p, 480p, 720p      |
| 854×480        | 360p, 480p            |
| 640×360        | 360p                  |
| 320×240 (sub-360p) | 360p (single fallback — small upscale tolerated over no HLS at all) |
| probe failed   | 360p (safe minimum)   |

Portrait videos use the same rule on the short side: a 1080×1920 portrait
renders all three variants; a 480×854 portrait renders 360p + 480p.

### Aspect ratio + orientation

The scale filter preserves aspect:

- Landscape: `scale=-2:<target>` — height = target, width auto (even).
- Portrait:  `scale=<target>:-2` — width = target, height auto (even).

Nothing is cropped, nothing is stretched, nothing rotates. The master
playlist advertises the actual computed output dimensions (e.g.
`RESOLUTION=1280x720` for landscape 720p, `RESOLUTION=720x1280` for
portrait 720p).

### ffmpeg invocation (per variant)

```
ffmpeg -loglevel error -y \
       -i <input.mp4> \
       -vf scale=-2:720 \                      # or scale=720:-2 for portrait
       -c:v libx264 -preset veryfast \
       -profile:v main -level 4.0 -pix_fmt yuv420p \
       -b:v 2500k -maxrate 2675k -bufsize 3750k \
       -g 120 -keyint_min 120 -sc_threshold 0 \
       -c:a aac -b:a 128k -ar 48000 -ac 2 \
       -hls_time 4 \
       -hls_list_size 0 \
       -hls_playlist_type vod \
       -hls_flags independent_segments \
       -hls_segment_filename <out>/720p/segment%03d.ts \
       -f hls \
       <out>/720p/index.m3u8
```

`-g 120` keeps a keyframe every 4 s @ 30 fps so segment boundaries
align with keyframes. `-sc_threshold 0` disables scene-change keyframes
that would misalign segments. The same invocation runs once per
eligible rung, with the bitrate triple and scale target swapped in.

After all variants are encoded, `BuildMasterPlaylist` writes the
top-level `master.m3u8`:

```
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-INDEPENDENT-SEGMENTS
#EXT-X-STREAM-INF:BANDWIDTH=900000,RESOLUTION=640x360,CODECS="avc1.4d401f,mp4a.40.2"
360p/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=1530000,RESOLUTION=854x480,CODECS="avc1.4d401f,mp4a.40.2"
480p/index.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=2630000,RESOLUTION=1280x720,CODECS="avc1.4d401f,mp4a.40.2"
720p/index.m3u8
```

Variants are listed in ascending bandwidth so hls.js picks the lowest
rung first and ramps up.

### Endpoints

| Method | Path                                          | Auth        | Notes                                    |
|--------|-----------------------------------------------|-------------|------------------------------------------|
| GET    | `/api/v1/hls/:video_id/master.m3u8`           | signed      | Public, signature in query string        |
| GET    | `/api/v1/hls/:video_id/:variant/index.m3u8`   | signed      | Variant playlist                         |
| GET    | `/api/v1/hls/:video_id/:variant/:segment`     | signed      | `.ts` segment, Range-aware via `http.ServeContent` |
| POST   | `/api/v1/admin/videos/:id/hls-url`            | Bearer      | Returns a signed master URL              |
| POST   | `/api/v1/site/videos/:id/hls-url`             | X-API-Key   | Same shape; restricted to public+ready   |

The signature uses the HLS scope (`hls:<video_id>:<expires>`), so a
signature minted for MP4 streaming won't unlock HLS and vice versa.

### Manifest URI rewriting

The manifest files on disk reference plain paths (`v1/index.m3u8`,
`segment000.ts`). The handler reads them and appends the request's
`?expires=…&signature=…` to every URI line on the way out, so the
player's subsequent fetches stay signed without any client-side
juggling. Comment lines (`#…`) are passed through unchanged.

This means **one signed URL gates an entire HLS playback session**: the
master, the variant playlist, and every segment all share the same
signature.

### Status semantics (`videos.hls_status`)

| Value         | Meaning                                                |
|---------------|--------------------------------------------------------|
| `pending`     | Fresh insert, conversion not started yet               |
| `processing`  | ffmpeg is running                                      |
| `ready`       | `master.m3u8` + segments on disk, `hls_ready_at` set   |
| `failed`      | ffmpeg or post-processing errored; `hls_error` populated |
| `skipped`     | Admin opted out (reserved — not yet emitted)           |

### Issuing a signed HLS URL

```bash
TOKEN=...   # admin JWT (or use the site X-API-Key endpoint)

curl -s -X POST http://localhost:8080/api/v1/admin/videos/$VIDEO_ID/hls-url \
  -H "Authorization: Bearer $TOKEN"
# → {
#     "hls_url": "http://localhost:8080/api/v1/hls/<id>/master.m3u8?expires=…&signature=…",
#     "expires_at": "2026-05-23T16:00:00Z",
#     "expires": 1716480000
#   }
```

If conversion hasn't finished, the issuer returns
`409 hls is still processing for this video` (vs. `404 not found` for
unknown ids), so the dashboard can poll.

### Verify HLS playback

```bash
# 1. Get the signed URL (above)
HLS_URL="http://localhost:8080/api/v1/hls/<id>/master.m3u8?expires=…&signature=…"

# 2. Fetch the master — should contain v1/index.m3u8?expires=…&signature=…
curl -s "$HLS_URL"

# 3. ffplay or VLC will play it directly
ffplay "$HLS_URL"     # or: vlc "$HLS_URL"

# 4. From a browser: Safari plays natively; Chrome/Firefox need hls.js
#    (the admin dashboard preview modal handles both transparently).
```

### MP4 vs HLS — which to use?

| Feature                    | MP4 (`/stream/:id`)        | HLS (`/hls/:id/master.m3u8`)  |
|----------------------------|----------------------------|-------------------------------|
| Available immediately      | Yes                        | After `hls_status='ready'`    |
| Range requests             | Yes (Phase 6)              | Yes (per segment)             |
| Adaptive bitrate           | No                         | Yes (Phase 14+ scaffold)      |
| Live / DVR / multi-track   | No                         | Yes (future)                  |
| CDN-friendly               | OK                         | Better (small, cacheable .ts) |
| Player support             | Native everywhere          | Native on Safari/iOS; hls.js elsewhere |

For Phase 13 the admin dashboard prefers HLS when `hls_status === 'ready'`
and falls back to the MP4 signed URL otherwise.

### Future roadmap (not in this build)

- libx264 / AAC re-encode for inputs that don't copy cleanly
- Multi-variant ladder (360p / 720p / 1080p)
- Queue worker so conversion isn't tied to the API process
- DRM / per-session keys
- CDN handoff via signed cookies

---

## Public video API (Phase 7)

This is the surface a WordPress (or any frontend) site uses to display
video grids. It only returns videos where `visibility='public'`,
`status='ready'`, and `deleted_at IS NULL` — enforced at the SQL layer,
so the public client can never widen those constraints.

Every response carries an absolute `thumbnail_url` pointing at
`/api/v1/thumbnails/:id`. Private internals (raw filenames, storage
paths, uploader IDs, MIME, byte sizes) are intentionally omitted.

### Public response shape

```json
{
  "id": "uuid",
  "title": "My First Video",
  "slug": "my-first-video",
  "description": "Test upload",
  "category": {"id": "uuid", "name": "Tutorials"},
  "thumbnail_url": "http://localhost:8080/api/v1/thumbnails/<id>",
  "duration_seconds": null,
  "views_count": 0,
  "created_at": "2026-05-23T...",
  "stream_available": true
}
```

### List — `GET /api/v1/videos`

```bash
curl -G http://localhost:8080/api/v1/videos \
  --data-urlencode 'page=1' \
  --data-urlencode 'limit=20' \
  --data-urlencode 'search=intro' \
  --data-urlencode 'category=<category-uuid>' \
  --data-urlencode 'sort=-created_at'
```

| Query param | Default       | Notes                                                                |
|-------------|---------------|----------------------------------------------------------------------|
| `page`      | `1`           | 1-indexed                                                            |
| `limit`     | `20`          | clamped to `[1, 100]`                                                |
| `search`    | —             | ILIKE on `title` and `slug`                                          |
| `category`  | —             | category UUID (slug filtering can be added later)                    |
| `sort`      | `-created_at` | whitelist: `title`, `-title`, `created_at`, `-created_at`, `views`, `-views` |

Response:

```json
{
  "data": [ ... PublicVideoResponse ... ],
  "pagination": {"page":1, "limit":20, "total":42, "total_pages":3}
}
```

### Detail — `GET /api/v1/videos/:slug`

Returns the same `PublicVideoResponse` as the list. Returns `404` for
private, soft-deleted, draft/processing/blocked, or unknown slugs — by
design, so the endpoint can't be used to enumerate private content.

```bash
curl http://localhost:8080/api/v1/videos/my-first-video
```

### Random feed — `GET /api/v1/videos/random`

```bash
# fresh random on every call
curl 'http://localhost:8080/api/v1/videos/random?limit=10'

# per-site daily-stable order
curl 'http://localhost:8080/api/v1/videos/random?limit=10&seed=site-a'
```

| Query param | Default | Notes                                                  |
|-------------|---------|--------------------------------------------------------|
| `limit`     | `20`    | clamped to `[1, 50]`                                   |
| `seed`      | —       | empty → `ORDER BY random()` (fresh on every call). Non-empty → stable for `<seed>:<UTC date>` |

The seed key is built as `<seed>:<YYYY-MM-DD>` (UTC), so:

- `site-a` gets one order today, a different one tomorrow
- `site-a` and `site-b` get different orders on the same day
- a site can be called from multiple regions and still see the same order
  for the same UTC day

Response:

```json
{
  "data": [ ... PublicVideoResponse ... ],
  "seed": "site-a:2026-05-23"
}
```

(`seed` is the actual key used; omitted when no seed was provided.)

Duplicates within a single response are not possible — the underlying
`ORDER BY md5(id || seed) LIMIT N` (or `ORDER BY random() LIMIT N`)
visits each row at most once.

### Promoting a video to public (admin)

To test the public API against an existing upload:

```bash
# get the admin JWT, then:
curl -X PUT http://localhost:8080/api/v1/admin/videos/$VIDEO_ID \
  -H "Authorization: Bearer $TOKEN" \
  -F "visibility=public"
```

Admin responses now also carry `thumbnail_url` (absolute), so the same
URL can be re-used by the admin panel preview.

### End-to-end smoke test (Phase 7)

```bash
# 1. log in
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/admin/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"change-me-please"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')

# 2. flip an existing upload to public
curl -s -X PUT http://localhost:8080/api/v1/admin/videos/$VIDEO_ID \
  -H "Authorization: Bearer $TOKEN" \
  -F "visibility=public"

# 3. public list
curl -s http://localhost:8080/api/v1/videos

# 4. seeded random (stable for the day)
curl -s 'http://localhost:8080/api/v1/videos/random?limit=10&seed=test-site'

# 5. detail by slug
curl -s http://localhost:8080/api/v1/videos/my-first-video

# 6. thumbnail (paste into a browser or save)
curl -o /tmp/t.jpg "http://localhost:8080/api/v1/thumbnails/$VIDEO_ID"
```

---

## Sites & site-protected API (Phase 8)

External frontends (WordPress installations, custom sites, etc.)
authenticate with **X-API-Key**, not the admin JWT. Each site owns one
domain and one API key; admins manage the roster through the JWT-only
`/admin/sites` endpoints, and the keys themselves are stored as SHA-256
hex digests.

Site video visibility is also controlled by `site_video_publications`.
A WordPress site only receives and can request playback URLs for videos
with a live publication row for that exact site whose status is
`approved` or `synced`. A globally `public` + `ready` video is not enough
on its own.

### Threat model & guarantees

- Plain API keys are returned **exactly once** — at create and at rotate.
  After that they exist only as SHA-256 hashes in the DB. There is no
  endpoint that reads them back.
- 256-bit random keys (`vsk_` + 64 hex chars) — collision-free in
  practice; offline brute force is infeasible.
- The X-API-Key middleware enforces a per-site domain check via Origin /
  Referer (subdomain suffix match, e.g. `example.com` covers
  `cdn.example.com`). Requests with neither header are allowed, so
  server-to-server callers still work. Browsers can't fake Origin.
- Disabled / suspended sites are rejected with `403`. Rotated or
  soft-deleted keys stop working immediately because the
  authentication lookup goes through the partial unique index on
  `api_key_hash WHERE deleted_at IS NULL`.

### Create a site (admin)

```bash
TOKEN=...   # admin JWT
curl -s -X POST http://localhost:8080/api/v1/admin/sites \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name":"Site A",
    "domain":"site-a.com",
    "status":"active",
    "daily_request_limit":10000
  }'
```

Response (`201 Created`):

```json
{
  "id": "uuid",
  "name": "Site A",
  "domain": "site-a.com",
  "status": "active",
  "daily_request_limit": 10000,
  "api_key": "vsk_a1b2c3d4...",     ← copy this NOW, you won't see it again
  "created_at": "...",
  "updated_at": "..."
}
```

> ⚠️ **Save `api_key` immediately**. Subsequent `GET` / `PUT` / list
> calls never return it. If you lose it, hit `/rotate-key`.

| Field                  | Notes                                                   |
|------------------------|---------------------------------------------------------|
| `name`                 | required                                                |
| `domain`               | required, unique among live sites (lower-cased on save) |
| `status`               | `active` (default) / `disabled` / `suspended`           |
| `daily_request_limit`  | optional; default `10000`. Stored only; enforcement is a later phase |

Failure responses:

| Condition                          | Status |
|------------------------------------|--------|
| Missing/invalid admin JWT          | `401`  |
| Domain collision                   | `409`  |
| Invalid name / domain / status / limit | `400` |

### Manage sites

```bash
# list (no key material returned)
curl -H "Authorization: Bearer $TOKEN" \
  'http://localhost:8080/api/v1/admin/sites?page=1&limit=20&status=active'

# fetch one
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/sites/$SITE_ID

# partial update
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Site A Prod","daily_request_limit":50000}' \
  http://localhost:8080/api/v1/admin/sites/$SITE_ID

# quick disable
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/sites/$SITE_ID/disable

# rotate the key (returns the new plain key once; old key dies)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/sites/$SITE_ID/rotate-key

# soft delete
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/sites/$SITE_ID
```

### Calling site endpoints (X-API-Key)

```bash
KEY="vsk_..."

# list
curl -H "X-API-Key: $KEY" \
  'http://localhost:8080/api/v1/site/videos?limit=10&category=$CATEGORY_ID'

# random feed — defaults seed to the site ID so each site gets its own
# stable daily order automatically
curl -H "X-API-Key: $KEY" \
  'http://localhost:8080/api/v1/site/videos/random?limit=10'

# override the seed if you need a per-page rotation
curl -H "X-API-Key: $KEY" \
  'http://localhost:8080/api/v1/site/videos/random?limit=10&seed=homepage-row-1'

# issue a signed stream URL (only works for public+ready videos)
curl -X POST -H "X-API-Key: $KEY" \
  http://localhost:8080/api/v1/site/videos/$VIDEO_ID/stream-url
```

The signed URL returned by `/site/videos/:id/stream-url` honours
`STREAM_URL_EXPIRATION_MINUTES` exactly like the admin endpoint, but
non-public or non-ready videos return `404` (so a site can't probe IDs
to find private content).

Site-API responses use the **same** `PublicVideoResponse` shape as
`/api/v1/videos`. The only differences: requires X-API-Key, and the
random feed auto-seeds with the site ID.

### Per-request access log

The middleware emits one line per site-authenticated request:

```
[SITE] site_id=… domain=… method=GET path=/api/v1/site/videos/random status=200 ip=… ua="…"
```

DB persistence + per-site quotas come in a later phase; the structured
shape above is what the analytics ingester will key on.

### Failure-response matrix

| Condition                                           | Status |
|-----------------------------------------------------|--------|
| Missing `X-API-Key`                                 | `401`  |
| Key not in DB / malformed shape                     | `401`  |
| Site `status='disabled'`                            | `403`  |
| Site `status='suspended'`                           | `403`  |
| Origin/Referer present but doesn't match site domain | `403`  |
| Site endpoint with private/non-ready video ID       | `404`  |

### Local-dev setup

Create a dev site whose `domain` is `localhost` (or `127.0.0.1` if
that's what you use). The Origin/Referer match treats those as exact
hosts, so a local WordPress at `http://localhost:8000` will be
accepted, and curl requests (no Origin) work too.

```bash
curl -s -X POST http://localhost:8080/api/v1/admin/sites \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Local Dev","domain":"localhost"}'
```

### End-to-end smoke test

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/admin/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"change-me-please"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["token"])')

# 1. create the site, capture the plain key
read SITE_ID API_KEY < <(curl -s -X POST http://localhost:8080/api/v1/admin/sites \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Site A","domain":"localhost"}' \
  | python3 -c 'import sys,json;d=json.load(sys.stdin);print(d["id"],d["api_key"])')

# 2. site list works
curl -i -H "X-API-Key: $API_KEY" http://localhost:8080/api/v1/site/videos

# 3. site random uses the site id as seed by default
curl -i -H "X-API-Key: $API_KEY" \
  'http://localhost:8080/api/v1/site/videos/random?limit=10'

# 4. rotate the key — the OLD one must stop working
NEW_KEY=$(curl -s -X POST http://localhost:8080/api/v1/admin/sites/$SITE_ID/rotate-key \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["api_key"])')

curl -i -H "X-API-Key: $API_KEY"  http://localhost:8080/api/v1/site/videos   # 401
curl -i -H "X-API-Key: $NEW_KEY" http://localhost:8080/api/v1/site/videos   # 200

# 5. disable — the new key now fails too
curl -s -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/admin/sites/$SITE_ID/disable

curl -i -H "X-API-Key: $NEW_KEY" http://localhost:8080/api/v1/site/videos   # 403

# 6. site-issued signed URL is restricted to public+ready (private → 404)
curl -i -X POST -H "X-API-Key: $NEW_KEY" \
  http://localhost:8080/api/v1/site/videos/$PRIVATE_VIDEO_ID/stream-url     # 404
```

---

## Database migrations

Migrations live in `migrations/` and follow the standard golang-migrate naming:

```
000001_create_admin_users.up.sql       # Phase 2
000001_create_admin_users.down.sql
000002_create_categories.up.sql        # Phase 4
000002_create_categories.down.sql
000003_create_videos.up.sql            # Phase 5
000003_create_videos.down.sql
000004_add_thumbnail_second_to_videos.up.sql    # Phase 5 — ffmpeg thumbnails
000004_add_thumbnail_second_to_videos.down.sql
000005_create_sites.up.sql                       # Phase 8 — multi-site API keys
000005_create_sites.down.sql
```

They are applied via the `migrate` service in `docker-compose.yml`,
which uses the official `migrate/migrate` image — no host-side tooling required.

```bash
docker compose run --rm migrate up                  # apply pending
docker compose run --rm migrate down 1              # roll back one
docker compose run --rm migrate version             # current version
docker compose run --rm migrate force <version>     # recovery only
```

Verify the admin_users table exists:

```bash
docker compose exec postgres \
    psql -U video_user -d video_storage -c "\d admin_users"
```

---

## Seeding the first admin

The `cmd/seed-admin` CLI reads three env vars:

| Variable             | Purpose                                |
|----------------------|----------------------------------------|
| `ADMIN_EMAIL`        | Email of the admin to create           |
| `ADMIN_PASSWORD`     | Plain password — hashed before insert  |
| `ADMIN_DISPLAY_NAME` | Human-readable display name            |

It connects with the same `DB_*` env as the API, checks for an existing
non-deleted admin with that email, and otherwise inserts a new
`super_admin` with status `active`. The plain password is hashed with
bcrypt (cost 12) — it never reaches the database.

```bash
docker compose exec api go run ./cmd/seed-admin
```

---

## Schema: `admin_users`

| Column                  | Type         | Notes                                     |
|-------------------------|--------------|-------------------------------------------|
| `id`                    | UUID         | PK, defaults to `gen_random_uuid()`       |
| `email`                 | TEXT         | unique (case-insensitive, ignores soft-deletes) |
| `password_hash`         | TEXT         | bcrypt hash only — never plain text       |
| `display_name`          | TEXT         |                                           |
| `role`                  | TEXT         | CHECK: `super_admin` / `admin` / `editor` — defaults to `admin` |
| `status`                | TEXT         | CHECK: `active` / `disabled` / `locked` — defaults to `active` |
| `last_login_at`         | TIMESTAMPTZ  | nullable                                  |
| `failed_login_attempts` | INTEGER      | defaults to 0; CHECK ≥ 0                  |
| `locked_until`          | TIMESTAMPTZ  | nullable                                  |
| `created_at`            | TIMESTAMPTZ  | defaults to `NOW()`                       |
| `updated_at`            | TIMESTAMPTZ  | defaults to `NOW()`                       |
| `deleted_at`            | TIMESTAMPTZ  | nullable — soft delete                    |

---

## Common Docker commands

```bash
# start in foreground (logs streaming)
docker compose up --build

# start in background
docker compose up -d --build

# tail logs
docker compose logs -f api
docker compose logs -f postgres

# stop containers (keep volumes)
docker compose down

# stop and wipe DB volume (fresh DB next boot — re-run migrations after!)
docker compose down -v

# open a shell inside the api container
docker compose exec api sh

# psql into the running Postgres
docker compose exec postgres psql -U video_user -d video_storage

# run Go tests inside the dev container
docker compose exec api go test ./...
```

---

## Running outside Docker (optional)

You still need a running Postgres. Either point at the dockerized one
(set `DB_HOST=localhost`) or use your own.

```bash
cp .env.example .env
# edit .env: set DB_HOST=localhost, set JWT_SECRET

go mod download
go run ./cmd/api
```

To run migrations from the host, install `migrate` and pass the same DSN:

```bash
migrate -path migrations \
    -database "postgres://video_user:video_password@localhost:5432/video_storage?sslmode=disable" \
    up
```

---

## Configuration

All configuration is loaded from environment variables (with `.env`
auto-loaded for local dev). See `.env.example` for the full list.

| Variable                       | Default            | Description                          |
|--------------------------------|--------------------|--------------------------------------|
| `APP_ENV`                      | `development`      | environment name                     |
| `API_PORT`                     | `8080`             | HTTP port                            |
| `DB_HOST`                      | `postgres`         | DB hostname (`localhost` off-Docker) |
| `DB_PORT`                      | `5432`             | DB port                              |
| `DB_USER`                      | `video_user`       | DB user                              |
| `DB_PASSWORD`                  | `video_password`   | DB password                          |
| `DB_NAME`                      | `video_storage`    | DB name                              |
| `DB_SSLMODE`                   | `disable`          | sslmode for pgx                      |
| `DB_MAX_CONNS`                 | `10`               | pgx pool max connections             |
| `JWT_SECRET`                   | (required)         | HS256 signing key                    |
| `JWT_EXPIRES_IN`               | `24h`              | token TTL                            |
| `LOGIN_MAX_ATTEMPTS`           | `5`                | wrong-password threshold             |
| `LOGIN_LOCK_DURATION_MINUTES`  | `15`               | lockout window                       |
| `ADMIN_EMAIL`                  | (none)             | seed-admin: email                    |
| `ADMIN_PASSWORD`               | (none)             | seed-admin: plain pwd (hashed before save) |
| `ADMIN_DISPLAY_NAME`           | (none)             | seed-admin: display name             |
| `VIDEO_STORAGE_PATH`           | `storage/private/videos` | private video dir              |
| `THUMBNAIL_STORAGE_PATH`       | `storage/private/thumbnails` | private thumbnail dir      |
| `MAX_VIDEO_UPLOAD_SIZE_MB`     | `2048`             | per-video upload cap                 |
| `MAX_THUMBNAIL_UPLOAD_SIZE_MB` | `10`               | reserved (thumbnails are ffmpeg-generated) |
| `STREAM_SIGNING_SECRET`        | (required)         | HMAC key for signed stream URLs      |
| `STREAM_URL_EXPIRATION_MINUTES`| `60`               | TTL of admin-issued stream URLs      |
| `ALLOWED_STREAM_DOMAINS`       | `""`               | comma-separated; empty / `*` = allow all |
| `PUBLIC_BASE_URL`              | `""`               | absolute base for signed URLs        |
| `ALLOWED_CORS_ORIGINS`         | `""`               | comma-separated Origin allowlist for browser callers (e.g. the admin dashboard) |

Do **not** commit `.env`. Only `.env.example` belongs in the repo.

---

## Next phase (not in this build)

- HLS transcoding + adaptive bitrate (multi-rendition output, master
  playlist served via `/api/v1/stream/...`)
- Queue worker / background processing pipeline (fill `duration_seconds`,
  generate poster frames, run transcode jobs)
- Background file-cleanup worker for soft-deleted videos
- View-count increments + analytics persistence (the `[STREAM]` access
  log already has the shape needed for this)
- S3 / object-storage backend behind the same `storage` interface
- CDN integration (signed-URL handoff to a downstream edge)
- WordPress integration
- Role-based authorisation (e.g. `RequireRole(RoleSuperAdmin)`) on top of `RequireAdmin`
- Refresh tokens / token revocation list

---

## Site category mapping contract

This repository implements only the stream-platform side of the two-sided WordPress category mapping workflow. The WordPress plugin lives outside this checkout and should consume the site API later.

Per site, WordPress categories imported through `POST /api/v1/site/sync/categories/import` are stored in `site_remote_categories`. Backend-to-WordPress mappings are stored in `site_category_mappings`. Remote term IDs are never global identifiers; every lookup is scoped by the authenticated `site_id` from `X-API-Key`.

Mapping rules:

- Many backend categories may map to one imported WordPress category.
- One backend category has at most one mapping row per site.
- Same-name suggestions are created disabled with `sync_mode=block`; they require explicit admin confirmation.
- `enabled=false`, missing remote category, or `sync_mode=block` prevents automatic publish behavior.
- A fully usable publish mapping requires `enabled=true`, a remote category, `confirmed_by_stream=true`, and `confirmed_by_wp=true`.

`GET /api/v1/site/sync/videos` includes category mapping metadata for each approved video:

```json
{
  "backend_category_id": "...",
  "backend_category_name": "Anime",
  "category_mapping_status": "mapped|unmapped|disabled|unconfirmed",
  "sync_mode": "publish|draft|block",
  "mapped_remote_category": {
    "id": "...",
    "remote_term_id": "12",
    "remote_slug": "cartoon",
    "remote_name": "การ์ตูน",
    "remote_taxonomy": "category"
  }
}
```

The future plugin should skip or draft videos with `category_mapping_status` other than `mapped`, and must not assign categories by same-name matching on its own.
