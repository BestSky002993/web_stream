# Production multi-stage build for the admin-dashboard.
# Uses Next.js `output: 'standalone'` for a minimal runtime image
# (~180 MB vs ~1 GB for a full node_modules copy).
#
# `NEXT_PUBLIC_*` env vars are baked into the client bundle at BUILD
# time. To point a production image at a different backend, pass it
# in via a build arg:
#
#   docker build --build-arg NEXT_PUBLIC_API_BASE_URL=https://api.prod.example.com \
#                -t admin-dashboard .

# ---------- Dependencies ----------
FROM node:20-alpine AS deps

WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci || npm install

# ---------- Build ----------
FROM node:20-alpine AS builder

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ARG NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
ENV NEXT_TELEMETRY_DISABLED=1

RUN npm run build

# ---------- Runtime ----------
FROM node:20-alpine AS runtime

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup --system --gid 1001 nodejs && \
    adduser  --system --uid 1001 nextjs

# `public/` is optional — only copy if it exists at build time.
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

CMD ["node", "server.js"]
