'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { Folder, LayoutDashboard, LogOut, Settings, Video, Globe, ShieldCheck } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { clearToken } from '@/lib/auth/storage';
import { cn } from '@/lib/utils/cn';
import { useCurrentAdmin } from './auth-guard';

const nav = [
	{ href: '/dashboard',  label: 'Dashboard',  icon: LayoutDashboard },
	{ href: '/videos',     label: 'Videos',     icon: Video },
	{ href: '/categories', label: 'Categories', icon: Folder },
	{ href: '/sites',      label: 'Sites',      icon: Globe },
	{ href: '/allowed-domains', label: 'Allowed Domains', icon: ShieldCheck },
	{ href: '/settings',   label: 'Settings',   icon: Settings },
];

export function Sidebar() {
	const pathname = usePathname();
	const router = useRouter();
	const admin = useCurrentAdmin();

	function logout() {
		clearToken();
		router.replace('/login');
	}

	return (
		<aside className="hidden w-60 shrink-0 flex-col border-r bg-card md:flex">
			<div className="flex h-14 items-center border-b px-5">
				<Link href="/dashboard" className="text-sm font-semibold tracking-tight">
					Video Storage
				</Link>
			</div>
			<nav className="flex-1 space-y-0.5 p-3">
				{nav.map((item) => {
					const Icon = item.icon;
					const active = pathname === item.href || pathname?.startsWith(item.href + '/');
					return (
						<Link
							key={item.href}
							href={item.href}
							className={cn(
								'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
								active
									? 'bg-accent text-accent-foreground'
									: 'text-muted-foreground hover:bg-accent hover:text-foreground',
							)}
						>
							<Icon className="h-4 w-4" />
							{item.label}
						</Link>
					);
				})}
			</nav>
			<div className="border-t p-3">
				<div className="mb-3 px-3">
					<div className="truncate text-sm font-medium">{admin.display_name}</div>
					<div className="truncate text-xs text-muted-foreground">{admin.email}</div>
				</div>
				<Button variant="ghost" size="sm" className="w-full justify-start" onClick={logout}>
					<LogOut className="mr-2 h-4 w-4" />
					Log out
				</Button>
			</div>
		</aside>
	);
}
