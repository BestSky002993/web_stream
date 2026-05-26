import { api } from './client';
import type { CategoryMappingListResponse, RemoteCategoryListResponse, SyncMode } from '@/lib/types';

export interface MappingSaveInput {
	backend_category_id: string;
	remote_category_id: string | null;
	enabled: boolean;
	sync_mode: SyncMode;
	confirmed_by_stream: boolean;
}

export function listRemoteCategories(siteId: string): Promise<RemoteCategoryListResponse> {
	return api<RemoteCategoryListResponse>(`/api/v1/admin/sites/${siteId}/remote-categories`);
}

export function listCategoryMappings(siteId: string): Promise<CategoryMappingListResponse> {
	return api<CategoryMappingListResponse>(`/api/v1/admin/sites/${siteId}/category-mappings`);
}

export function saveCategoryMappings(siteId: string, mappings: MappingSaveInput[]): Promise<CategoryMappingListResponse> {
	return api<CategoryMappingListResponse>(`/api/v1/admin/sites/${siteId}/category-mappings`, { method: 'PUT', json: { mappings } });
}

export function suggestCategoryMappings(siteId: string): Promise<CategoryMappingListResponse> {
	return api<CategoryMappingListResponse>(`/api/v1/admin/sites/${siteId}/category-mappings/suggest`, { method: 'POST' });
}
