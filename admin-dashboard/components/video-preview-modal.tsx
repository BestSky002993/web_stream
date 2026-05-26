'use client';

import * as React from 'react';
import { toast } from 'sonner';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Skeleton } from '@/components/ui/skeleton';
import { Badge } from '@/components/ui/badge';
import { issueHLSURL, issueStreamURL } from '@/lib/api/videos';
import type { HLSStatus } from '@/lib/types';

interface VideoPreviewModalProps {
	videoId: string | null;
	title?: string;
	/** Pass the current HLS status so the modal can choose HLS first when ready. */
	hlsStatus?: HLSStatus;
	onOpenChange: (open: boolean) => void;
}

type Source = { kind: 'hls' | 'mp4'; url: string };

/**
 * Admin video preview.
 *
 * Picks the best playback source on open:
 *   1. HLS (signed master.m3u8) when `hlsStatus === 'ready'`
 *      - native HLS in Safari/iOS via `<video src>`
 *      - hls.js (dynamic import) elsewhere
 *   2. MP4 signed URL otherwise (e.g. HLS still processing)
 *
 * Tears down on close: pauses, drops src, calls .load(), and destroys
 * the hls.js instance so the network request to the backend ends
 * immediately.
 */
export function VideoPreviewModal({ videoId, title, hlsStatus, onOpenChange }: VideoPreviewModalProps) {
	const [source, setSource] = React.useState<Source | null>(null);
	const [loading, setLoading] = React.useState(false);
	const [error, setError] = React.useState<string | null>(null);

	const isOpen = videoId !== null;
	const videoRef = React.useRef<HTMLVideoElement | null>(null);
	const hlsInstance = React.useRef<{ destroy: () => void } | null>(null);

	// Resolve a source whenever the modal opens for a new video.
	React.useEffect(() => {
		if (!videoId) {
			setSource(null);
			setError(null);
			return;
		}
		let cancelled = false;
		setLoading(true);
		setError(null);

		(async () => {
			try {
				if (hlsStatus === 'ready') {
					const { hls_url } = await issueHLSURL(videoId);
					if (cancelled) return;
					setSource({ kind: 'hls', url: hls_url });
				} else {
					const { stream_url } = await issueStreamURL(videoId);
					if (cancelled) return;
					setSource({ kind: 'mp4', url: stream_url });
				}
			} catch (err) {
				const msg = err instanceof Error ? err.message : 'Could not load video';
				if (cancelled) return;
				setError(msg);
				toast.error(msg);
			} finally {
				if (!cancelled) setLoading(false);
			}
		})();

		return () => {
			cancelled = true;
		};
	}, [videoId, hlsStatus]);

	// Attach the source to the <video> element. For HLS we use the
	// browser's native support when available; otherwise dynamically
	// import hls.js so users who only ever preview MP4 don't pay the
	// bundle cost.
	React.useEffect(() => {
		const video = videoRef.current;
		if (!video || !source) return;

		// Always destroy any previous hls.js instance first.
		if (hlsInstance.current) {
			hlsInstance.current.destroy();
			hlsInstance.current = null;
		}

		if (source.kind === 'mp4') {
			video.src = source.url;
			return;
		}

		// HLS path
		if (video.canPlayType('application/vnd.apple.mpegurl')) {
			video.src = source.url;
			return;
		}

		let cancelled = false;
		import('hls.js')
			.then((mod) => {
				if (cancelled) return;
				const Hls = mod.default;
				if (Hls.isSupported()) {
					const inst = new Hls();
					inst.loadSource(source.url);
					inst.attachMedia(video);
					hlsInstance.current = inst;
				} else {
					// No HLS support at all — fall back to native src and
					// let the browser show its own error.
					video.src = source.url;
				}
			})
			.catch((err) => {
				if (!cancelled) {
					setError('Could not load HLS player');
					toast.error(`HLS player failed: ${err instanceof Error ? err.message : String(err)}`);
				}
			});

		return () => {
			cancelled = true;
		};
	}, [source]);

	function handleOpenChange(next: boolean) {
		if (!next) {
			// Tear down on close.
			const video = videoRef.current;
			if (video) {
				try { video.pause(); } catch { /* ignore */ }
				video.removeAttribute('src');
				try { video.load(); } catch { /* ignore */ }
			}
			if (hlsInstance.current) {
				hlsInstance.current.destroy();
				hlsInstance.current = null;
			}
			setSource(null);
			setError(null);
		}
		onOpenChange(next);
	}

	return (
		<Dialog open={isOpen} onOpenChange={handleOpenChange}>
			<DialogContent className="max-w-3xl gap-0 overflow-hidden p-0">
				<DialogHeader className="border-b p-4">
					<DialogTitle className="flex items-center gap-2 text-base">
						<span>{title || 'Preview'}</span>
						{source && (
							<Badge variant={source.kind === 'hls' ? 'success' : 'muted'}>
								{source.kind === 'hls' ? 'HLS' : 'MP4'}
							</Badge>
						)}
					</DialogTitle>
					<DialogDescription>Signed URL — expires automatically.</DialogDescription>
				</DialogHeader>
				<div className="flex aspect-video w-full items-center justify-center bg-black">
					{loading && <Skeleton className="h-full w-full" />}
					{!loading && error && <div className="p-6 text-sm text-destructive">{error}</div>}
					{!loading && !error && source && (
						<video
							ref={videoRef}
							key={source.url}
							controls
							autoPlay
							playsInline
							preload="metadata"
							className="h-full w-full"
						/>
					)}
				</div>
			</DialogContent>
		</Dialog>
	);
}
