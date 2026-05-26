'use client';

import * as React from 'react';
import { useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import useSWR from 'swr';
import { swrFetcher } from '@/lib/api/client';
import { getToken, clearToken, isExpired } from '@/lib/auth/storage';
import type { Admin } from '@/lib/types';

interface AuthGuardProps {
	children: React.ReactNode;
}

// Tiny context — pages can read the current admin without an extra fetch.
export const AuthContext = React.createContext<Admin | null>(null);

export function useCurrentAdmin(): Admin {
	const ctx = React.useContext(AuthContext);
	if (!ctx) throw new Error('useCurrentAdmin used outside AuthGuard');
	return ctx;
}

/**
 * Wraps protected admin pages. Redirects to /login if:
 *   - no token in localStorage
 *   - token decoded `exp` is in the past
 *   - GET /api/v1/admin/me returns 401 / fails
 *
 * While verifying, renders a minimal placeholder rather than the
 * protected content (avoids flashing the dashboard at unauthenticated
 * users).
 */
export function AuthGuard({ children }: AuthGuardProps) {
	const router = useRouter();
	const pathname = usePathname();

	const token = typeof window !== 'undefined' ? getToken() : null;
	const localExpired = token ? isExpired(token) : true;
	const hasToken = !!token && !localExpired;

	const { data, error, isLoading } = useSWR<Admin>(hasToken ? '/api/v1/admin/me' : null, swrFetcher, {
		shouldRetryOnError: false,
	});

	useEffect(() => {
		if (!hasToken) {
			clearToken();
			const next = encodeURIComponent(pathname || '/dashboard');
			router.replace(`/login?next=${next}`);
		}
	}, [hasToken, router, pathname]);

	useEffect(() => {
		if (error) {
			clearToken();
			const next = encodeURIComponent(pathname || '/dashboard');
			router.replace(`/login?next=${next}`);
		}
	}, [error, router, pathname]);

	if (!hasToken || isLoading || error || !data) {
		return (
			<div className="flex min-h-screen items-center justify-center">
				<div className="text-sm text-muted-foreground">Loading…</div>
			</div>
		);
	}

	return <AuthContext.Provider value={data}>{children}</AuthContext.Provider>;
}
