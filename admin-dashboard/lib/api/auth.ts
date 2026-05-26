import { api } from './client';
import type { Admin, LoginResponse } from '@/lib/types';

export function login(email: string, password: string): Promise<LoginResponse> {
	return api<LoginResponse>('/api/v1/admin/auth/login', {
		method: 'POST',
		json: { email, password },
		auth: false,
	});
}

export function fetchMe(): Promise<Admin> {
	return api<Admin>('/api/v1/admin/me');
}
