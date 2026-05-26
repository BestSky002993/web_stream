# Video Storage — Admin Dashboard

Next.js + TypeScript + Tailwind dashboard for managing the
[video-storage-api](../video-storage-api) backend.

A pure client of the existing API: same JWT login as Phase 3, same
multipart upload as Phase 5, same signed stream URLs as Phase 6, same
site-key management as Phase 8. Nothing new on the server.

## Project layout

```
admin-dashboard/
├── app/
│   ├── layout.tsx                   ← root shell + sonner Toaster
│   ├── globals.css                  ← Tailwind + CSS variables (light/dark)
│   ├── login/page.tsx               ← POST /api/v1/admin/auth/login
│   └── (protected)/
│       ├── layout.tsx               ← AuthGuard + Sidebar + Topbar
│       ├── dashboard/page.tsx       ← stat cards + quick actions
│       ├── videos/page.tsx          ← list + search + filters + actions
│       ├── videos/new/page.tsx      ← multipart upload + progress
│       ├── videos/[id]/page.tsx     ← edit + preview
│       ├── categories/page.tsx     ← CRUD (inline dialog)
│       ├── sites/page.tsx          ← CRUD + one-shot key reveal + rotate
│       └── settings/page.tsx       ← read-only config + current admin
├── components/
│   ├── auth-guard.tsx               ← reads JWT, calls /admin/me, redirects to /login
│   ├── sidebar.tsx                  ← desktop nav + logout
│   ├── topbar.tsx                   ← mobile nav + logout
│   ├── page-header.tsx              ← <h1> + actions
│   ├── status-badge.tsx             ← coloured pill per backend status
│   ├── empty-state.tsx
│   ├── confirm-dialog.tsx           ← typed destructive confirms
│   ├── upload-dropzone.tsx          ← drag-drop, MIME+size check
│   ├── video-preview-modal.tsx      ← signed-URL <video> player
│   ├── api-key-modal.tsx            ← one-shot "save this NOW" modal
│   └── ui/                          ← shadcn-style primitives
│       ├── button, input, label, card, badge, table, skeleton,
│       ├── dialog (Radix), dropdown-menu (Radix), select (Radix)
├── lib/
│   ├── api/
│   │   ├── client.ts                ← fetch wrapper (auth, 401 redirect)
│   │   ├── auth.ts, videos.ts, categories.ts, sites.ts
│   ├── auth/storage.ts              ← getToken / setToken / decodeJWT / isExpired
│   ├── types/index.ts               ← every response shape
│   └── utils/cn.ts, format.ts
├── tailwind.config.ts, postcss.config.mjs, next.config.mjs, tsconfig.json
├── package.json, .env.example, .gitignore, README.md
```

## Requirements

- Node.js 20+
- pnpm / npm / yarn
- A running [video-storage-api](../video-storage-api) backend
  (default `http://localhost:8080`)

## Install + run (host Node.js)

```bash
cd admin-dashboard
cp .env.example .env.local
# edit .env.local if your backend isn't on http://localhost:8080
npm install
npm run dev
# → open http://localhost:3000
```

The dev server defaults to port 3000. The backend's
`ALLOWED_STREAM_DOMAINS` only affects public streaming requests — the
admin API doesn't check Origin, so no CORS tweaks are usually needed.

## Install + run (Docker)

The dashboard ships a standalone `docker-compose.yml` plus a separate
`Dockerfile.dev` (hot reload) and `Dockerfile` (production multi-stage,
Next.js standalone output).

```bash
cd admin-dashboard
docker compose up --build
# → open http://localhost:3000
```

That's it. Source-code changes trigger Next's HMR via the `.:/app`
bind mount, and `node_modules` + `.next` inside the container survive
because the compose file anonymously mounts them over the bind.

### Stop / restart

```bash
docker compose down            # stop the dashboard container
docker compose up --build      # rebuild + start (after package.json changes)
docker compose logs -f         # tail logs
```

### Configuring the API base URL

`NEXT_PUBLIC_*` env vars are baked into the **client bundle**. For the
dev image that doesn't matter — the bundle is rebuilt on every code
change anyway — but you need to give the right value:

| Where your backend is              | `NEXT_PUBLIC_API_BASE_URL`           |
|------------------------------------|--------------------------------------|
| Same host, default API ports       | `http://localhost:8080` *(default)*  |
| Remote / staging API               | `https://api.staging.example.com`    |

```bash
# Inline override:
NEXT_PUBLIC_API_BASE_URL=http://192.168.1.50:8080 docker compose up --build

# Or in a .env at the project root:
echo 'NEXT_PUBLIC_API_BASE_URL=http://192.168.1.50:8080' > .env
docker compose up --build
```

**Cross-container DNS doesn't help here.** Even if you wire the
dashboard and the API into the same Docker network and set
`NEXT_PUBLIC_API_BASE_URL=http://api:8080`, the value travels into the
JavaScript that runs in the *user's browser* — which has no idea what
`api` resolves to. The URL must be reachable from the browser, so
`http://localhost:8080` (or a real DNS name) is correct.

### Running both projects together

The dashboard and the API publish their own ports on the host, so
they can run side-by-side without sharing a Docker network:

```bash
# Terminal A — the backend
( cd ../video-storage-api && docker compose up --build )

# Terminal B — the dashboard
( cd ../admin-dashboard   && docker compose up --build )

# → http://localhost:8080  (Go API)
# → http://localhost:3000  (admin dashboard)
```

### Production build

**Run all production commands from the repo root (`web_stream/`), not
from this directory.** The full stack (API + dashboard + Postgres +
nginx) is orchestrated by the root [`docker-compose.prod.yml`](../docker-compose.prod.yml)
and driven by the helper scripts in [`../scripts/`](../scripts):

```bash
cd web_stream
./scripts/prod-init-env.sh           # creates .env.production from the example
$EDITOR .env.production               # fill in secrets
./scripts/prod-up.sh                  # build + start everything
./scripts/prod-migrate.sh             # apply DB migrations
```

See [`../README.md`](../README.md) for the full operational guide.

### Building just the dashboard image

```bash
docker build \
  --build-arg NEXT_PUBLIC_API_BASE_URL=https://api.example.com \
  --build-arg NEXT_PUBLIC_BASE_PATH=/admin \
  -f Dockerfile.prod \
  -t admin-dashboard:prod .

docker run --rm -p 3000:3000 admin-dashboard:prod
```

`Dockerfile.prod` is a multi-stage build that emits a Next.js
standalone server (~180 MB image), runs as a non-root user, binds to
`0.0.0.0:3000`, and bakes `NEXT_PUBLIC_*` at build time (rebuild to
change them).

When the dashboard is mounted behind a reverse proxy under a path
(e.g. `/admin`), pass `NEXT_PUBLIC_BASE_PATH=/admin` so Next.js
prefixes its routes and static asset URLs accordingly. Leave it empty
to serve the dashboard at the host root.

## Configure

| Variable                   | Default                  | Notes                       |
|----------------------------|--------------------------|-----------------------------|
| `NEXT_PUBLIC_API_BASE_URL` | `http://localhost:8080`  | absolute base of the Go API |

`NEXT_PUBLIC_*` env vars are baked into the client at build time. To
change the backend after `npm run build`, set the var and rebuild.

## How auth works

1. `POST /api/v1/admin/auth/login` returns `{ token, expires_in, admin }`.
2. The token is saved in `localStorage` under `vsa.token`.
3. Every request through `lib/api/client.ts` attaches
   `Authorization: Bearer <token>`.
4. `AuthGuard` (on the `(protected)` route group):
   - reads the token, decodes the JWT, checks `exp` locally
   - calls `GET /api/v1/admin/me` via SWR
   - redirects to `/login?next=…` on missing / expired / 401
5. Any `401` response anywhere clears the token + bounces to login.

localStorage is acceptable for this phase per the spec. A future phase
can swap it for an HttpOnly cookie — the call sites stay the same.

## Logging in

The dashboard uses the same admin user as the backend. If you ran the
seed-admin CLI:

```bash
docker compose exec api go run ./cmd/seed-admin
```

then sign in with the credentials from `.env` (`ADMIN_EMAIL`,
`ADMIN_PASSWORD`).

## Uploading a video

1. **Videos → Upload**.
2. Drop a `.mp4` / `.webm` / `.mov` (client-side MIME + size check).
3. Fill in title, slug (auto-derived from title), category, visibility,
   thumbnail second.
4. Submit. The page uses `XMLHttpRequest` so you get a real upload
   progress bar; the backend extracts the thumbnail with ffmpeg and
   returns the full record. You're redirected to `/videos/[id]`.

## Managing sites

1. **Sites → New site**.
2. After submit, the backend returns the plain `api_key` exactly once.
   The dashboard immediately shows the `<APIKeyModal>` with a copy
   button and a "save it now" warning.
3. To rotate: `Sites → ⋯ → Rotate API key`. A confirmation dialog
   explains the old key dies immediately. On confirm, the new plain key
   is shown — once.
4. To kill access without losing the site record: `⋯ → Disable`.
5. To free up the domain/key for reuse: `⋯ → Delete` (soft delete).

The plain key is **never** displayed again after creation/rotation, by
design (and by backend) — the dashboard mirrors that.

## Preview a video

`Videos → ⋯ → Preview` (or from the edit page). The modal:

1. Calls `POST /api/v1/admin/videos/:id/stream-url`.
2. Sets that URL on a `<video controls autoplay>`.
3. On close, drops `src` + calls `.load()` so the byte stream stops
   immediately (no buffered memory hanging around).

## Error handling

- Any `401` from the API → token cleared, redirect to `/login`.
- All other backend errors surface as `sonner` toasts using the
  `error` field from the JSON envelope.
- Network failures show "Could not reach backend" without crashing the
  page.
- Tables show skeletons while loading; empty results show a typed
  empty state with a primary action.

## Testing locally end-to-end

```bash
# 1. Run the backend
( cd ../video-storage-api && docker compose up -d --build && docker compose run --rm migrate up )

# 2. Run the dashboard
cp .env.example .env.local
npm install
npm run dev

# 3. In the browser:
#    a) http://localhost:3000/login → sign in with the seed admin
#    b) Categories → New category ("Tutorials" / "tutorials")
#    c) Videos → Upload → pick a .mp4 → submit
#    d) Open the upload, click "Preview" → modal plays via signed URL
#    e) Sites → New site → copy the api_key
#    f) Plug api_key into the video-storage-embed WordPress plugin
```

## Build for production

```bash
npm run build
npm run start
```

By default Next.js serves on port 3000. Put it behind a reverse proxy
(nginx, Caddy, Traefik) that terminates TLS — both this dashboard and
the Go backend.

## Known limitations

- localStorage JWT — to be replaced by an HttpOnly cookie in a future
  phase (the API client + AuthGuard interfaces won't change).
- No HLS playback yet — preview uses progressive MP4/WebM/MOV directly
  from the backend's Range-aware endpoint.
- No multi-admin role UI yet.
- No infinite scroll on lists — paginated with prev/next.
- The mobile sidebar is a simple top-drawer, no swipe gestures.
