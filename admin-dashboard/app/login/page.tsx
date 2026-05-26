import { Suspense } from 'react';
import { LoginForm } from '@/components/login-form';

/**
 * The login route stays a server component so Next.js can statically
 * pre-render the chrome. The form itself is a client component because
 * it uses `useSearchParams()` to honour the `?next=` redirect target —
 * wrapping it in <Suspense> satisfies the App Router's CSR-bailout
 * boundary requirement (and keeps `next build` from erroring out with
 * "useSearchParams() should be wrapped in a suspense boundary").
 */
export default function LoginPage() {
	return (
		<div className="flex min-h-screen items-center justify-center bg-muted p-4">
			<Suspense fallback={<LoginFallback />}>
				<LoginForm />
			</Suspense>
		</div>
	);
}

function LoginFallback() {
	return (
		<div className="w-full max-w-sm rounded-lg border bg-card p-6 text-sm text-muted-foreground shadow-sm">
			Loading…
		</div>
	);
}
