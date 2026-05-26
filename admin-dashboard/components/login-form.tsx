'use client';

import { useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { toast } from 'sonner';
import { Loader2 } from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { login } from '@/lib/api/auth';
import { setToken } from '@/lib/auth/storage';

/**
 * Client-side login form.
 *
 * `useSearchParams()` opts every component that calls it (or any of
 * its server-component parents) into client-side rendering. To keep
 * `app/login/page.tsx` a server component AND avoid the
 * "useSearchParams() should be wrapped in a suspense boundary"
 * production-build error, this hook is isolated here behind a
 * `<Suspense>` boundary the parent page sets up.
 */
export function LoginForm() {
	const router = useRouter();
	const search = useSearchParams();
	const next = search.get('next') || '/dashboard';

	const [email, setEmail] = useState('');
	const [password, setPassword] = useState('');
	const [pending, setPending] = useState(false);

	async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		if (!email || !password) return;
		setPending(true);
		try {
			const res = await login(email, password);
			setToken(res.token);
			toast.success(`Welcome, ${res.admin.display_name}`);
			router.replace(safeNext(next));
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Login failed';
			toast.error(msg);
		} finally {
			setPending(false);
		}
	}

	return (
		<Card className="w-full max-w-sm">
			<CardHeader>
				<CardTitle>Sign in</CardTitle>
				<CardDescription>Use your admin email and password.</CardDescription>
			</CardHeader>
			<CardContent>
				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="space-y-1.5">
						<Label htmlFor="email">Email</Label>
						<Input
							id="email"
							type="email"
							autoComplete="email"
							placeholder="admin@example.com"
							value={email}
							onChange={(e) => setEmail(e.target.value)}
							required
							autoFocus
						/>
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="password">Password</Label>
						<Input
							id="password"
							type="password"
							autoComplete="current-password"
							value={password}
							onChange={(e) => setPassword(e.target.value)}
							required
						/>
					</div>
					<Button type="submit" className="w-full" disabled={pending}>
						{pending ? (
							<>
								<Loader2 className="h-4 w-4 animate-spin" /> Signing in…
							</>
						) : (
							'Sign in'
						)}
					</Button>
				</form>
			</CardContent>
		</Card>
	);
}

/** Refuse `next` values that try to escape this app. */
function safeNext(s: string): string {
	if (!s.startsWith('/') || s.startsWith('//')) return '/dashboard';
	return s;
}
