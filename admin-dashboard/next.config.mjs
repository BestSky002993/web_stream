/** @type {import('next').NextConfig} */

// Optional reverse-proxy mount point. Empty → dashboard at the root
// (default; works for `npm run dev`). Set to e.g. `/admin` at build
// time to serve the dashboard behind that path. NEXT_PUBLIC_* vars
// are baked in at `next build`, so this is a build-time switch.
const basePath = process.env.NEXT_PUBLIC_BASE_PATH || '';

const nextConfig = {
	reactStrictMode: true,

	// Emit a minimal `.next/standalone` build that the production
	// Dockerfile copies without dragging the full node_modules tree
	// into the runtime image.
	output: 'standalone',

	// Only set basePath when explicitly configured — otherwise `next
	// dev` would 404 the root because routes would be under the prefix.
	...(basePath ? { basePath } : {}),

	// Backend's `/api/v1/thumbnails/:id` returns image bytes. Without an
	// allowlist next/image would refuse external hosts — but we use plain
	// <img> tags everywhere for simplicity, so no remotePatterns entry
	// is needed yet.
};

export default nextConfig;
