import { api } from './client';
import type { AllowedDomain, AllowedDomainListResponse, AllowedDomainStatus, AllowedDomainType } from '@/lib/types';

export function listAllowedDomains(params: { page?: number; limit?: number; search?: string; type?: string; status?: string } = {}): Promise<AllowedDomainListResponse> {
	const q = new URLSearchParams();
	for (const [k, v] of Object.entries(params)) {
		if (v !== undefined && v !== null && v !== '') q.set(k, String(v));
	}
	const qs = q.toString();
	return api<AllowedDomainListResponse>(`/api/v1/admin/allowed-domains${qs ? '?' + qs : ''}`);
}

export interface AllowedDomainInput {
	domain: string;
	type: AllowedDomainType;
	status?: AllowedDomainStatus;
	notes?: string;
}

export function createAllowedDomain(input: AllowedDomainInput): Promise<AllowedDomain> {
	return api<AllowedDomain>('/api/v1/admin/allowed-domains', { method: 'POST', json: input });
}

export type UpdateAllowedDomainInput = Partial<AllowedDomainInput>;

export function updateAllowedDomain(id: string, input: UpdateAllowedDomainInput): Promise<AllowedDomain> {
	return api<AllowedDomain>(`/api/v1/admin/allowed-domains/${id}`, { method: 'PUT', json: input });
}

export function disableAllowedDomain(id: string): Promise<AllowedDomain> {
	return updateAllowedDomain(id, { status: 'disabled' });
}

export function deleteAllowedDomain(id: string): Promise<void> {
	return api<void>(`/api/v1/admin/allowed-domains/${id}`, { method: 'DELETE' });
}
