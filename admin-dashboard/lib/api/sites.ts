import { api } from './client';
import type { Site, SiteListResponse } from '@/lib/types';

export function listSites(params: { page?: number; limit?: number; search?: string; status?: string } = {}): Promise<SiteListResponse> {
	const q = new URLSearchParams();
	if (params.page) q.set('page', String(params.page));
	if (params.limit) q.set('limit', String(params.limit));
	if (params.search) q.set('search', params.search);
	if (params.status) q.set('status', params.status);
	const qs = q.toString();
	return api<SiteListResponse>(`/api/v1/admin/sites${qs ? '?' + qs : ''}`);
}

export interface CreateSiteInput {
	name: string;
	domain: string;
	status?: 'active' | 'disabled' | 'suspended';
	daily_request_limit?: number;
}

export function createSite(input: CreateSiteInput): Promise<Site> {
	return api<Site>('/api/v1/admin/sites', { method: 'POST', json: input });
}

export type UpdateSiteInput = Partial<CreateSiteInput>;

export function updateSite(id: string, input: UpdateSiteInput): Promise<Site> {
	return api<Site>(`/api/v1/admin/sites/${id}`, { method: 'PUT', json: input });
}

export function disableSite(id: string): Promise<Site> {
	return api<Site>(`/api/v1/admin/sites/${id}/disable`, { method: 'POST' });
}

export function rotateSiteKey(id: string): Promise<Site> {
	return api<Site>(`/api/v1/admin/sites/${id}/rotate-key`, { method: 'POST' });
}

export function deleteSite(id: string): Promise<void> {
	return api<void>(`/api/v1/admin/sites/${id}`, { method: 'DELETE' });
}
