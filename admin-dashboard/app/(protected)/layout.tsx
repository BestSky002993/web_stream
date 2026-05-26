import { AuthGuard } from '@/components/auth-guard';
import { Sidebar } from '@/components/sidebar';
import { Topbar } from '@/components/topbar';

export default function ProtectedLayout({ children }: { children: React.ReactNode }) {
	return (
		<AuthGuard>
			<div className="flex min-h-screen">
				<Sidebar />
				<div className="flex min-w-0 flex-1 flex-col">
					<Topbar />
					<main className="flex-1 overflow-x-hidden p-4 md:p-6">{children}</main>
				</div>
			</div>
		</AuthGuard>
	);
}
