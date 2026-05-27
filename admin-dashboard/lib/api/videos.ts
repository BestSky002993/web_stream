import { api, apiBase } from './client';
import { getToken } from '@/lib/auth/storage';
import type { BulkDistributionResponse, SignedHLSURL, SignedStreamURL, SiteDistributionResponse, SiteDistributionRow, SyncMode, Video, VideoListResponse } from '@/lib/types';

export interface VideoListParams {
	page?: number;
	limit?: number;
	search?: string;
	category_id?: string;
	status?: string;
	visibility?: string;
	not_approved_site_id?: string;
	sort?: string;
}

export function listVideos(params: VideoListParams = {}): Promise<VideoListResponse> {
	const q = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== null && v !== '') q.set(k, String(v));
	}
	const qs = q.toString();
	return api<VideoListResponse>(`/api/v1/admin/videos${qs ? '?' + qs : ''}`);
}

export function getVideo(id: string): Promise<Video> {
	return api<Video>(`/api/v1/admin/videos/${id}`);
}

export interface UploadVideoInput {
	title: string;
	slug: string;
	description?: string;
	category_id: string;
	category_ids?: string[];
	visibility?: 'public' | 'private';
	thumbnail_second?: number;
	video: File;
	onProgress?: (pct: number) => void;
	signal?: AbortSignal;
}

/**
 * Upload a video via multipart/form-data using XHR (so we can report
 * progress). Returns the created Video record from the API on success.
 */
export function uploadVideo(input: UploadVideoInput): Promise<Video> {
	return new Promise((resolve, reject) => {
		const fd = new FormData();
		fd.append('title', input.title);
		fd.append('slug', input.slug);
		if (input.description) fd.append('description', input.description);
		fd.append('category_id', input.category_id);
		for (const id of input.category_ids ?? []) fd.append('category_ids', id);
		if (input.visibility) fd.append('visibility', input.visibility);
		if (input.thumbnail_second !== undefined) fd.append('thumbnail_second', String(input.thumbnail_second));
		fd.append('video', input.video);

		const xhr = new XMLHttpRequest();
		xhr.open('POST', apiBase() + '/api/v1/admin/videos');
		const token = getToken();
		if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
		xhr.responseType = 'text';

		xhr.upload.addEventListener('progress', (e) => {
			if (e.lengthComputable && input.onProgress) {
				input.onProgress(Math.round((e.loaded / e.total) * 100));
			}
		});

		xhr.addEventListener('load', () => {
			const body = parseJSON(xhr.responseText);
			if (xhr.status >= 200 && xhr.status < 300) {
				resolve(body as Video);
			} else {
				const msg = body && typeof body === 'object' && 'error' in (body as any)
					? String((body as any).error)
					: `HTTP ${xhr.status}`;
				reject(new Error(msg));
			}
		});
		xhr.addEventListener('error', () => reject(new Error('Network error during upload')));
		xhr.addEventListener('abort', () => reject(new Error('Upload aborted')));

		input.signal?.addEventListener('abort', () => xhr.abort());
		xhr.send(fd);
	});
}

export interface UpdateVideoInput {
	title?: string;
	slug?: string;
	description?: string;
	category_id?: string;
	category_ids?: string[];
	status?: string;
	visibility?: 'public' | 'private';
	thumbnail_second?: number;
}

/** Multipart PUT — empty strings mean "no change" on the backend. */
export function updateVideo(id: string, input: UpdateVideoInput): Promise<Video> {
	const fd = new FormData();
	for (const [k, v] of Object.entries(input)) {
		if (k === 'category_ids' && Array.isArray(v)) {
			for (const id of v) fd.append('category_ids', id);
			continue;
		}
		if (v !== undefined && v !== null && v !== '') fd.append(k, String(v));
	}
	return api<Video>(`/api/v1/admin/videos/${id}`, {
		method: 'PUT',
		body: fd,
	});
}

export function deleteVideo(id: string): Promise<void> {
	return api<void>(`/api/v1/admin/videos/${id}`, { method: 'DELETE' });
}

export function issueStreamURL(id: string): Promise<SignedStreamURL> {
	return api<SignedStreamURL>(`/api/v1/admin/videos/${id}/stream-url`, { method: 'POST' });
}

export function issueHLSURL(id: string): Promise<SignedHLSURL> {
	return api<SignedHLSURL>(`/api/v1/admin/videos/${id}/hls-url`, { method: 'POST' });
}

export interface SaveSiteDistributionInput {
	site_id: string;
	status: 'approved' | 'blocked' | 'draft';
	sync_mode: SyncMode;
}

export function getSiteDistribution(id: string): Promise<SiteDistributionResponse> {
	return api<SiteDistributionResponse>(`/api/v1/admin/videos/${id}/site-distribution`);
}

export function saveSiteDistribution(id: string, input: SaveSiteDistributionInput): Promise<SiteDistributionRow> {
	return api<SiteDistributionRow>(`/api/v1/admin/videos/${id}/site-distribution`, { method: 'PUT', json: input });
}

export interface BulkDistributionInput {
	video_ids: string[];
	status: 'approved' | 'draft' | 'blocked';
	sync_mode: SyncMode;
	unmapped_behavior: 'skip' | 'draft';
}

export function bulkSaveSiteDistribution(siteId: string, input: BulkDistributionInput): Promise<BulkDistributionResponse> {
	return api<BulkDistributionResponse>(`/api/v1/admin/sites/${siteId}/video-distribution/bulk`, { method: 'POST', json: input });
}

function parseJSON(s: string): unknown {
	try { return JSON.parse(s); } catch { return undefined; }
}
