'use client';

import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { LogOut, Menu } from 'lucide-react';
import { useState } from 'react';
import { usePathname } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { clearToken } from '@/lib/auth/storage';
import { cn } from '@/lib/utils/cn';
import { useCurrentAdmin } from './auth-guard';

const mobileNav = [
	{ href: '/dashboard',  label: 'Dashboard' },
	{ href: '/videos',     label: 'Videos' },
	{ href: '/categories', label: 'Categories' },
	{ href: '/sites',      label: 'Sites' },
	{ href: '/settings',   label: 'Settings' },
];

export function Topbar() {
	const admin = useCurrentAdmin();
	const router = useRouter();
	const pathname = usePathname();
	const [open, setOpen] = useState(false);

	function logout() {
		clearToken();
		router.replace('/login');
	}

	return (
		<header className="sticky top-0 z-30 flex h-14 items-center gap-3 border-b bg-background px-4 md:px-6">
			<Button variant="ghost" size="icon" className="md:hidden" onClick={() => setOpen((o) => !o)} aria-label="Toggle navigation">
				<Menu className="h-5 w-5" />
			</Button>
			<div className="flex-1" />
			<div className="hidden text-right text-xs sm:block">
				<div className="font-medium">{admin.display_name}</div>
				<div className="text-muted-foreground">{admin.role}</div>
			</div>
			<Button variant="ghost" size="icon" onClick={logout} aria-label="Log out">
				<LogOut className="h-4 w-4" />
			</Button>

			{open && (
				<nav className="absolute inset-x-0 top-14 flex flex-col gap-1 border-b bg-card p-3 shadow-md md:hidden">
					{mobileNav.map((item) => {
						const active = pathname === item.href || pathname?.startsWith(item.href + '/');
						return (
							<Link
								key={item.href}
								href={item.href}
								onClick={() => setOpen(false)}
								className={cn(
									'rounded-md px-3 py-2 text-sm font-medium',
									active ? 'bg-accent text-accent-foreground' : 'text-muted-foreground hover:bg-accent',
								)}
							>
								{item.label}
							</Link>
						);
					})}
				</nav>
			)}
		</header>
	);
}
