import { api } from './client';
import type { Category, CategoryListResponse } from '@/lib/types';

export function listCategories(params: { page?: number; limit?: number; search?: string; status?: string } = {}): Promise<CategoryListResponse> {
	const q = new URLSearchParams();
	if (params.page) q.set('page', String(params.page));
	if (params.limit) q.set('limit', String(params.limit));
	if (params.search) q.set('search', params.search);
	if (params.status) q.set('status', params.status);
	const qs = q.toString();
	return api<CategoryListResponse>(`/api/v1/admin/categories${qs ? '?' + qs : ''}`);
}

export interface CreateCategoryInput {
	name: string;
	slug: string;
	description?: string;
	status?: 'active' | 'disabled';
	sort_order?: number;
}

export function createCategory(input: CreateCategoryInput): Promise<Category> {
	return api<Category>('/api/v1/admin/categories', { method: 'POST', json: input });
}

export type UpdateCategoryInput = Partial<CreateCategoryInput>;

export function updateCategory(id: string, input: UpdateCategoryInput): Promise<Category> {
	return api<Category>(`/api/v1/admin/categories/${id}`, { method: 'PUT', json: input });
}

export function deleteCategory(id: string): Promise<void> {
	return api<void>(`/api/v1/admin/categories/${id}`, { method: 'DELETE' });
}
