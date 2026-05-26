import { api } from './client';
import type { PublicationListResponse, PublicationStatus, SiteVideoPublication } from '@/lib/types';

export interface UpsertPublicationInput {
	status: PublicationStatus;
	sync_mode?: string | null;
	remote_post_id?: string | null;
	remote_slug?: string | null;
	category_mapping?: string | null;
	tags?: string[];
	notes?: string | null;
	last_synced_at?: string | null;
}

export function listSitePublications(siteId: string, params: { page?: number; limit?: number; status?: string } = {}): Promise<PublicationListResponse> {
	const q = new URLSearchParams();
	if (params.page) q.set('page', String(params.page));
	if (params.limit) q.set('limit', String(params.limit));
	if (params.status) q.set('status', params.status);
	const qs = q.toString();
	return api<PublicationListResponse>(`/api/v1/admin/sites/${siteId}/publications${qs ? '?' + qs : ''}`);
}

export function listVideoPublications(videoId: string): Promise<PublicationListResponse> {
	return api<PublicationListResponse>(`/api/v1/admin/videos/${videoId}/publications?limit=100`);
}

export function upsertPublication(siteId: string, videoId: string, input: UpsertPublicationInput): Promise<SiteVideoPublication> {
	return api<SiteVideoPublication>(`/api/v1/admin/sites/${siteId}/videos/${videoId}/publication`, { method: 'PUT', json: input });
}

export function approvePublication(id: string): Promise<SiteVideoPublication> {
	return api<SiteVideoPublication>(`/api/v1/admin/publications/${id}/approve`, { method: 'POST' });
}

export function disablePublication(id: string): Promise<SiteVideoPublication> {
	return api<SiteVideoPublication>(`/api/v1/admin/publications/${id}/disable`, { method: 'POST' });
}

export function resyncPublication(id: string): Promise<SiteVideoPublication> {
	return api<SiteVideoPublication>(`/api/v1/admin/publications/${id}/resync`, { method: 'POST' });
}

export function removePublication(id: string): Promise<void> {
	return api<void>(`/api/v1/admin/publications/${id}`, { method: 'DELETE' });
}
