# Video storage platform

Production commands are run from this root `web_stream/` folder only, using Docker Compose. Do not run host `npm` or `go` commands for production deployment.

`tripbooking.net` is the video storage and streaming platform. External WordPress sites connect to it with a Site API Key and an allowed domain entry.

Admin dashboard:

```text
https://tripbooking.net/control-9x7k-admin-panel/login
```

## VPS requirements

- Linux VPS with Docker Engine and Docker Compose plugin.
- Public IPv4 address.
- Ports `80/tcp` and `443/tcp` open in the VPS firewall.
- Enough disk for PostgreSQL plus original videos, thumbnails, and HLS renditions.
- At least 2 GB RAM for small deployments. Use more for concurrent uploads and HLS conversion.

## Docker install

Use Docker's official install method for your distribution. On Ubuntu this is typically:

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

## Cloudflare DNS and TLS

DNS:

```text
A @ -> VPS_IP
```

Cloudflare SSL/TLS:

```text
SSL/TLS encryption mode = Full (strict)
Always Use HTTPS = optional, nginx already redirects port 80 to 443
WebSockets = on
Cache Rules = bypass /api/* unless you have a deliberate streaming cache plan
```

Use a Cloudflare Origin Certificate or another valid certificate on the VPS. Put the files here:

```text
deploy/certs/fullchain.pem
deploy/certs/privkey.pem
```

Certificate files are ignored by git. The nginx container mounts them read-only at `/etc/nginx/certs`.

## Production env

Create the production env:

```bash
cp .env.production.example .env.production
$EDITOR .env.production
```

Required production values:

```env
PUBLIC_BASE_URL=https://tripbooking.net
ADMIN_BASE_PATH=/control-9x7k-admin-panel
ALLOWED_STREAM_DOMAINS=tripbooking.net
ALLOWED_CORS_ORIGINS=https://tripbooking.net
HTTP_PORT=80
HTTPS_PORT=443
```

Generate secrets with Docker only:

```bash
docker run --rm alpine:3.20 sh -c "apk add --no-cache openssl >/dev/null && openssl rand -hex 32"
```

Set unique values for:

```env
DB_PASSWORD=<strong database password>
JWT_SECRET=<hex secret, at least 32 chars>
STREAM_SIGNING_SECRET=<hex secret, at least 32 chars>
ADMIN_EMAIL=<initial admin email>
ADMIN_PASSWORD=<strong initial admin password>
ADMIN_DISPLAY_NAME=<initial admin display name>
```

Do not use `*` in `ALLOWED_CORS_ORIGINS` or `ALLOWED_STREAM_DOMAINS`.

## Deploy

Start or rebuild the stack:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

Run migrations:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml run --rm migrate up
```

Seed the first admin:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml run --rm seed-admin
```

Open:

```text
https://tripbooking.net/control-9x7k-admin-panel/login
```

## Services

| Service | Purpose | Public exposure |
| --- | --- | --- |
| `nginx` | HTTPS reverse proxy, redirects HTTP to HTTPS, serves `/api/`, dashboard, streaming, HLS, thumbnails | `80`, `443` |
| `api` | Compiled Go API, `APP_ENV=production`, `GIN_MODE=release` | Internal only |
| `dashboard` | Next.js standalone production server, not `next dev` | Internal only |
| `postgres` | PostgreSQL 16 persistent database | Internal only |
| `migrate` | One-shot migration runner | Internal only |
| `seed-admin` | One-shot admin seeder | Internal only |

PostgreSQL is not published to the host. Uploaded videos, thumbnails, and HLS output live in the private `storage_data_prod` Docker volume and are only served through authorized API routes.

## WordPress integration

External WordPress sites do not run on `tripbooking.net`. They install the Video Storage Embed plugin and connect to:

```text
API Base URL = https://tripbooking.net
Site API Key = key generated in the admin dashboard
```

For each WordPress site:

1. In `Sites`, create a site for the WordPress host, for example `example-wp-site.com`.
2. Copy the generated Site API Key. It is shown once.
3. In `Allowed Domains`, add the WordPress host, choose `both`, and keep it `active`.
4. In WordPress, set API Base URL to `https://tripbooking.net` and paste the Site API Key.

Controlled sync flow:

```text
Tripbooking admin uploads video
Tripbooking admin creates Site and Allowed Domain
Tripbooking admin approves that video for that specific Site
WordPress plugin -> GET /api/v1/site/sync/videos with X-API-Key
WordPress plugin creates/updates local Videos CPT posts
Viewer opens WP post
WordPress plugin -> POST /api/v1/site/videos/:id/stream-url or /hls-url with X-API-Key
Browser player -> signed MP4 or signed HLS URL
```

No approval record means the WordPress site receives zero videos, even when the video is globally `public` and `ready`.

Validation checklist:

- Missing `X-API-Key` on `/api/v1/site/videos` returns `401`.
- Disabled or deleted site returns `403` or `401`.
- WordPress host missing from `Allowed Domains` does not get CORS approval and stream hotlink checks fail.
- Video list and sync endpoints return public, ready videos only when `site_video_publications.status` is `approved` or `synced` for that API key's site.
- Thumbnail URLs load only from allowed domains.
- MP4 stream URLs include `expires` and `signature`.
- HLS master, variant playlists, and segments include `expires` and `signature`.
- Site API keys are used by the WordPress server or plugin settings flow only. Do not render them into public page HTML.

The WordPress plugin source is not present in this checkout, so plugin PHP internals cannot be audited or changed here.

### Controlled publishing

Admins control WordPress distribution with per-site publication rows:

```text
site_video_publications(site_id, video_id, status)
```

Statuses:

```text
draft     = not visible to WordPress
approved  = visible to this site's sync/list API
synced    = visible and already imported by WordPress
disabled  = hidden from this site and playback URL issuance
failed    = sync attempted and failed; hidden until approved/resynced
```

Dashboard controls:

- Video detail page -> Site publishing: choose target Site, approve/disable, category mapping, tags, notes.
- Sites -> Approved videos: view approved videos, remote post ID, last sync time, approve, disable, resync, remove.

Site API endpoints:

```text
GET  /api/v1/site/videos
GET  /api/v1/site/videos/random
GET  /api/v1/site/sync/videos
GET  /api/v1/site/sync/categories
POST /api/v1/site/sync/videos/:id/status
POST /api/v1/site/videos/:id/stream-url
POST /api/v1/site/videos/:id/hls-url
```

All site endpoints require `X-API-Key`. Listing, sync, and site-issued playback URLs only work for `approved` or `synced` publications owned by that key's site.

### WordPress sync contract

The plugin should keep the Site API Key server-side only. It should not print it into page HTML, enqueue it into JavaScript, or store it in public post meta.

Admin sync page:

- Test connection.
- Pull approved videos from `GET /api/v1/site/sync/videos`.
- Preview imported/updated videos before writing posts.
- Sync selected videos.
- Sync all approved videos.
- Dry run.
- Last sync status.

When syncing:

- Create/update the theme's native Videos CPT.
- Store API `video_id` in protected post meta.
- Store duration and publication sync metadata.
- Store thumbnail URL or sideload the image only if the URL host exactly matches the configured API Base URL.
- Map API category to WP video category.
- Map API tags to WP tags.
- Map actors to the WP actors taxonomy if the theme exposes one.
- Store `video_id`, not a signed stream URL.
- Never save signed HLS/MP4 URLs permanently.
- After successful import, call `POST /api/v1/site/sync/videos/:id/status` with `status=synced`, `remote_post_id`, and `remote_slug`.

Category mapping:

- Fetch API categories from `GET /api/v1/site/sync/categories`.
- Provide an admin mapping UI: API Category -> WP Video Category.
- Provide settings for API Tags -> WP Tags and API Actors -> WP Actors taxonomy when available.
- Recommended setting: `Auto-create missing taxonomies: yes/no`.

Playback:

- The synced WP post stores API `video_id`.
- The player calls a WordPress AJAX/REST endpoint with a nonce.
- WordPress server validates nonce and uses the server-side Site API Key to request a signed HLS or MP4 URL.
- The browser receives only the short-lived signed playback URL.

## Streaming validation

Issue URLs from the dashboard or Site API, then verify:

```bash
curl -I "https://tripbooking.net/api/v1/stream/<video-id>?expires=<unix>&signature=<sig>"
curl -I -H "Range: bytes=0-1023" "https://tripbooking.net/api/v1/stream/<video-id>?expires=<unix>&signature=<sig>"
curl -i "https://tripbooking.net/api/v1/hls/<video-id>/master.m3u8?expires=<unix>&signature=<sig>"
```

Expected:

- MP4 full response: `200`, `Accept-Ranges: bytes`.
- MP4 range response: `206`, `Content-Range`.
- Unsigned MP4 response: `401`.
- HLS master playlist: `application/vnd.apple.mpegurl`.
- Variant playlist and `.ts` segments load with the same signed query parameters. Future `.m4s`/`init.mp4` HLS segment files are also accepted by the API path validator.
- Safari can play HLS directly.
- Chromium/Firefox players use `hls.js`.

## Operations

Show services:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml ps
```

Follow logs:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f
```

Follow selected logs:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f api nginx dashboard
```

Restart safely:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build
```

Stop without deleting data:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml down
```

Check migrations:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml run --rm migrate version
```

## Rotating secrets

JWT secret:

1. Generate a new `JWT_SECRET`.
2. Update `.env.production`.
3. Restart with `docker compose --env-file .env.production -f docker-compose.prod.yml up -d --build`.
4. Existing admin sessions are invalidated.

Stream signing secret:

1. Generate a new `STREAM_SIGNING_SECRET`.
2. Update `.env.production`.
3. Restart the stack.
4. Existing signed MP4 and HLS URLs stop working. Players must request fresh URLs.

Site API key:

1. Use the dashboard Site key rotation action.
2. Update the WordPress plugin setting with the new key.
3. The old key stops authenticating.

## Backups

Back up PostgreSQL:

```bash
docker compose --env-file .env.production -f docker-compose.prod.yml exec -T postgres pg_dump -U video_user -d video_storage -Fc > backups/video_storage_$(date +%F).dump
```

Back up named volumes from the VPS:

```bash
docker run --rm -v web_stream_postgres_data_prod:/volume -v "$PWD/backups:/backup" alpine:3.20 tar czf /backup/postgres_data_$(date +%F).tgz -C /volume .
docker run --rm -v web_stream_storage_data_prod:/volume -v "$PWD/backups:/backup" alpine:3.20 tar czf /backup/storage_data_$(date +%F).tgz -C /volume .
```

Storage backups must include originals, thumbnails, and HLS output:

```text
/app/storage/private/videos
/app/storage/private/thumbnails
/app/storage/private/hls
```

Keep off-server encrypted backups and test restore on a separate machine before relying on them.

## Security checklist

- No wildcard production CORS. `ALLOWED_CORS_ORIGINS` must list explicit origins.
- No wildcard stream domains. `ALLOWED_STREAM_DOMAINS` must list explicit hosts.
- MP4 and HLS media bytes require signed URLs.
- Site API endpoints require `X-API-Key`.
- Site API listing/sync/playback issuance requires per-site publication approval.
- Allowed Domains controls browser CORS and stream hotlink domains.
- PostgreSQL is internal only and has no published port.
- Production uses compiled Go, no Air.
- Production dashboard uses Next.js standalone, no `next dev`.
- Private storage is in a Docker named volume, not directly served by nginx.
- Upload filenames are randomized and path sanitized.
- Upload MIME is detected server-side and checked against an allowlist.
- Admin JWTs have expiration and are cleared by the dashboard on `401`.
- Nginx sets TLS, HSTS, nosniff, frame, referrer, and permissions headers.
- Dashboard login redirects reject external `next` URLs.
- Site API keys must not be rendered into public frontend markup.
- WordPress AJAX endpoints must validate nonces, sanitize API metadata before inserting posts, escape output, and avoid thumbnail SSRF by accepting media only from the configured API base URL.

## Future scale items

- Move video/HLS storage to object storage or a dedicated persistent disk when volume size grows.
- Add a queue-backed HLS worker pool for high upload concurrency.
- Add CDN cache rules for signed HLS segments after confirming token TTL and cache behavior.
- Add automated certificate renewal if not using long-lived Cloudflare Origin Certificates.
- Add scheduled backup jobs and restore drills.
