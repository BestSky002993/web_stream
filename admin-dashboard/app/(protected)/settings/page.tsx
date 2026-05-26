'use client';

import { PageHeader } from '@/components/page-header';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { useCurrentAdmin } from '@/components/auth-guard';
import { apiBase } from '@/lib/api/client';
import { Button } from '@/components/ui/button';
import Link from 'next/link';

export default function SettingsPage() {
	const admin = useCurrentAdmin();

	return (
		<>
			<PageHeader title="Settings" description="Read-only configuration for this dashboard instance." />

			<div className="grid gap-4 lg:grid-cols-2">
				<Card>
					<CardHeader><CardTitle>Backend connection</CardTitle></CardHeader>
					<CardContent className="space-y-3 text-sm">
						<Row label="API base URL" value={<code className="text-xs">{apiBase()}</code>} />
						<Row label="Env variable" value={<code className="text-xs">NEXT_PUBLIC_API_BASE_URL</code>} />
						<p className="text-xs text-muted-foreground">
							Change in <code>.env.local</code> and restart <code>next dev</code>.
						</p>
					</CardContent>
				</Card>

				<Card>
					<CardHeader><CardTitle>Signed in as</CardTitle></CardHeader>
					<CardContent className="space-y-3 text-sm">
						<Row label="Email" value={admin.email} />
						<Row label="Display name" value={admin.display_name} />
						<Row label="Role" value={<code className="text-xs">{admin.role}</code>} />
						<Row label="Status" value={<code className="text-xs">{admin.status}</code>} />
						<Row label="Admin ID" value={<code className="text-xs">{admin.id}</code>} />
					</CardContent>
				</Card>

				<Card>
					<CardHeader><CardTitle>Allowed domains</CardTitle></CardHeader>
					<CardContent className="space-y-3 text-sm">
						<p className="text-muted-foreground">
							Manage hosts allowed for browser CORS and stream hotlink checks.
						</p>
						<Button asChild>
							<Link href="/allowed-domains">Open Allowed Domains</Link>
						</Button>
					</CardContent>
				</Card>
			</div>
		</>
	);
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
	return (
		<div className="flex items-center justify-between gap-3 border-b py-2 last:border-0">
			<div className="text-muted-foreground">{label}</div>
			<div className="font-medium">{value}</div>
		</div>
	);
}
