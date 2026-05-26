'use client';

import Link from 'next/link';
import useSWR from 'swr';
import { Folder, Globe, Video } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { PageHeader } from '@/components/page-header';
import { swrFetcher } from '@/lib/api/client';
import { useCurrentAdmin } from '@/components/auth-guard';
import type { CategoryListResponse, SiteListResponse, VideoListResponse } from '@/lib/types';

export default function DashboardPage() {
	const admin = useCurrentAdmin();
	const { data: videos, isLoading: lv } = useSWR<VideoListResponse>('/api/v1/admin/videos?limit=1', swrFetcher);
	const { data: cats,   isLoading: lc } = useSWR<CategoryListResponse>('/api/v1/admin/categories?limit=1', swrFetcher);
	const { data: sites,  isLoading: ls } = useSWR<SiteListResponse>('/api/v1/admin/sites?limit=1', swrFetcher);

	return (
		<>
			<PageHeader
				title={`Hello, ${admin.display_name}`}
				description="Overview of your video library and connected sites."
			/>

			<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				<StatCard
					icon={<Video className="h-5 w-5" />}
					label="Videos"
					value={videos?.pagination.total}
					loading={lv}
					href="/videos"
				/>
				<StatCard
					icon={<Folder className="h-5 w-5" />}
					label="Categories"
					value={cats?.pagination.total}
					loading={lc}
					href="/categories"
				/>
				<StatCard
					icon={<Globe className="h-5 w-5" />}
					label="Sites"
					value={sites?.pagination.total}
					loading={ls}
					href="/sites"
				/>
			</div>

			<div className="mt-8 grid gap-4 lg:grid-cols-2">
				<Card>
					<CardHeader>
						<CardTitle>Quick actions</CardTitle>
					</CardHeader>
					<CardContent className="flex flex-wrap gap-2">
						<Button asChild>
							<Link href="/videos/new">Upload a video</Link>
						</Button>
						<Button asChild variant="outline">
							<Link href="/categories">Manage categories</Link>
						</Button>
						<Button asChild variant="outline">
							<Link href="/sites">Add a site</Link>
						</Button>
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<CardTitle>About this dashboard</CardTitle>
					</CardHeader>
					<CardContent className="space-y-2 text-sm text-muted-foreground">
						<p>This UI is a client to the video-storage-api backend. Authentication uses your admin JWT, stored in <code>localStorage</code>.</p>
						<p>Stream URLs in the preview modal are signed on demand and expire automatically.</p>
					</CardContent>
				</Card>
			</div>
		</>
	);
}

interface StatCardProps {
	icon: React.ReactNode;
	label: string;
	value: number | undefined;
	loading: boolean;
	href: string;
}

function StatCard({ icon, label, value, loading, href }: StatCardProps) {
	return (
		<Link href={href} className="block transition-transform hover:-translate-y-0.5">
			<Card>
				<CardContent className="flex items-center gap-4 p-6">
					<div className="rounded-md bg-muted p-3">{icon}</div>
					<div className="flex-1">
						<div className="text-sm text-muted-foreground">{label}</div>
						{loading ? (
							<Skeleton className="mt-1 h-7 w-12" />
						) : (
							<div className="text-2xl font-semibold tabular-nums">{value ?? 0}</div>
						)}
					</div>
				</CardContent>
			</Card>
		</Link>
	);
}
