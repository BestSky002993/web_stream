'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';
import useSWR from 'swr';
import { toast } from 'sonner';
import { Ban, CheckCircle2, ChevronLeft, ChevronRight, Copy, Eye, FileDown, Globe2, Loader2, MoreHorizontal, Pencil, Plus, Search, Send, ShieldOff, Trash2, Video as VideoIcon } from 'lucide-react';
import { PageHeader } from '@/components/page-header';
import { StatusBadge } from '@/components/status-badge';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { VideoPreviewModal } from '@/components/video-preview-modal';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Skeleton } from '@/components/ui/skeleton';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { swrFetcher } from '@/lib/api/client';
import { bulkSaveSiteDistribution, deleteVideo, getSiteDistribution, issueStreamURL, saveSiteDistribution, updateVideo } from '@/lib/api/videos';
import { listSites } from '@/lib/api/sites';
import { formatDate } from '@/lib/utils/format';
import type { CategoryListResponse, SiteDistributionResponse, SiteDistributionRow, SiteListResponse, SyncMode, Video, VideoListResponse } from '@/lib/types';
import { mutate } from 'swr';

const PAGE_SIZE = 20;

export default function VideosPage() {
	const [page, setPage] = useState(1);
	const [search, setSearch] = useState('');
	const [searchInput, setSearchInput] = useState('');
	const [visibility, setVisibility] = useState<string>('');
	const [categoryId, setCategoryId] = useState<string>('');

	const qs = new URLSearchParams();
	qs.set('page', String(page));
	qs.set('limit', String(PAGE_SIZE));
	qs.set('sort', '-created_at');
	if (search) qs.set('search', search);
	if (visibility) qs.set('visibility', visibility);
	if (categoryId) qs.set('category_id', categoryId);

	const key = `/api/v1/admin/videos?${qs.toString()}`;
	const { data, isLoading } = useSWR<VideoListResponse>(key, swrFetcher);
	const { data: cats } = useSWR<CategoryListResponse>('/api/v1/admin/categories?limit=100', swrFetcher);

	const [previewId, setPreviewId] = useState<string | null>(null);
	const [previewTitle, setPreviewTitle] = useState<string | undefined>(undefined);
	const [previewHLSStatus, setPreviewHLSStatus] = useState<Video['hls']['status'] | undefined>(undefined);
	const [deleteTarget, setDeleteTarget] = useState<Video | null>(null);
	const [deletePending, setDeletePending] = useState(false);
	const [distributionTarget, setDistributionTarget] = useState<{ video: Video; action: DistributionAction } | null>(null);
	const [selectedIds, setSelectedIds] = useState<string[]>([]);
	const [bulkOpen, setBulkOpen] = useState(false);
	const [bulkDefaults, setBulkDefaults] = useState<{ status: 'approved' | 'draft' | 'blocked'; syncMode: SyncMode }>({ status: 'approved', syncMode: 'publish' });

	function applySearch(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setPage(1);
		setSearch(searchInput);
	}

	async function toggleVisibility(v: Video) {
		const next = v.visibility === 'public' ? 'private' : 'public';
		try {
			await updateVideo(v.id, { visibility: next });
			toast.success(`Set to ${next}`);
			mutate((k) => typeof k === 'string' && k.startsWith('/api/v1/admin/videos'));
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Update failed');
		}
	}

	async function copyStreamURL(v: Video) {
		try {
			const r = await issueStreamURL(v.id);
			await navigator.clipboard.writeText(r.stream_url);
			toast.success('Stream URL copied (expires automatically)');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Could not generate URL');
		}
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		setDeletePending(true);
		try {
			await deleteVideo(deleteTarget.id);
			toast.success('Video deleted');
			setDeleteTarget(null);
			mutate((k) => typeof k === 'string' && k.startsWith('/api/v1/admin/videos'));
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Delete failed');
		} finally {
			setDeletePending(false);
		}
	}

	const items = data?.data ?? [];
	const pagination = data?.pagination;
	const selectedOnPage = items.filter((v) => selectedIds.includes(v.id)).length;
	const allPageSelected = items.length > 0 && selectedOnPage === items.length;

	function toggleSelected(id: string, checked: boolean) {
		setSelectedIds((current) => checked ? Array.from(new Set([...current, id])) : current.filter((v) => v !== id));
	}

	function togglePage(checked: boolean) {
		const pageIds = items.map((v) => v.id);
		setSelectedIds((current) => checked ? Array.from(new Set([...current, ...pageIds])) : current.filter((id) => !pageIds.includes(id)));
	}

	function openBulk(status: 'approved' | 'draft' | 'blocked', syncMode: SyncMode) {
		setBulkDefaults({ status, syncMode });
		setBulkOpen(true);
	}

	return (
		<>
			<PageHeader
				title="Videos"
				description="All uploaded videos, including private and unpublished ones."
				actions={
					<Button asChild>
						<Link href="/videos/new">
							<Plus className="h-4 w-4" /> Upload
						</Link>
					</Button>
				}
			/>

			<div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
				<form onSubmit={applySearch} className="relative w-full sm:max-w-xs">
					<Search className="pointer-events-none absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
					<Input
						type="search"
						placeholder="Search title or slug…"
						value={searchInput}
						onChange={(e) => setSearchInput(e.target.value)}
						className="pl-8"
					/>
				</form>

				<Select value={visibility || 'all'} onValueChange={(v) => { setVisibility(v === 'all' ? '' : v); setPage(1); }}>
					<SelectTrigger className="sm:w-40"><SelectValue placeholder="Visibility" /></SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All visibility</SelectItem>
						<SelectItem value="public">Public</SelectItem>
						<SelectItem value="private">Private</SelectItem>
					</SelectContent>
				</Select>

				<Select value={categoryId || 'all'} onValueChange={(v) => { setCategoryId(v === 'all' ? '' : v); setPage(1); }}>
					<SelectTrigger className="sm:w-52"><SelectValue placeholder="Category" /></SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All categories</SelectItem>
						{cats?.data.map((c) => (
							<SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
						))}
					</SelectContent>
				</Select>
			</div>

			{selectedIds.length > 0 && (
				<div className="mb-4 flex flex-wrap items-center gap-2 rounded-md border bg-muted px-3 py-2 text-sm">
					<span className="font-medium">{selectedIds.length} selected</span>
					<Button size="sm" onClick={() => openBulk('approved', 'publish')}><Send className="h-4 w-4" /> Publish selected to Site</Button>
					<Button size="sm" variant="outline" onClick={() => openBulk('approved', 'draft')}><CheckCircle2 className="h-4 w-4" /> Approve selected to Site</Button>
					<Button size="sm" variant="outline" onClick={() => openBulk('draft', 'draft')}><FileDown className="h-4 w-4" /> Draft selected to Site</Button>
					<Button size="sm" variant="outline" onClick={() => openBulk('blocked', 'block')}><Ban className="h-4 w-4" /> Block selected from Site</Button>
					<Button size="sm" variant="ghost" onClick={() => setSelectedIds([])}>Clear</Button>
				</div>
			)}

			{isLoading ? (
				<TableSkeleton rows={6} />
			) : items.length === 0 ? (
				<EmptyState
					icon={<VideoIcon className="h-8 w-8" />}
					title="No videos yet"
					description="Upload your first video to see it here."
					action={
						<Button asChild>
							<Link href="/videos/new">Upload a video</Link>
						</Button>
					}
				/>
			) : (
				<div className="rounded-lg border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead className="w-10">
									<input type="checkbox" aria-label="Select current page" checked={allPageSelected} onChange={(e) => togglePage(e.currentTarget.checked)} />
								</TableHead>
								<TableHead className="w-32">Thumbnail</TableHead>
								<TableHead>Title</TableHead>
								<TableHead>Category</TableHead>
									<TableHead>Status</TableHead>
									<TableHead>Visibility</TableHead>
									<TableHead>Sites</TableHead>
									<TableHead className="text-right">Views</TableHead>
								<TableHead>Created</TableHead>
								<TableHead className="w-10" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{items.map((v) => (
								<TableRow key={v.id}>
									<TableCell>
										<input type="checkbox" aria-label={`Select ${v.title}`} checked={selectedIds.includes(v.id)} onChange={(e) => toggleSelected(v.id, e.currentTarget.checked)} />
									</TableCell>
									<TableCell>
										{/* eslint-disable-next-line @next/next/no-img-element */}
										<img
											src={v.thumbnail_url}
											alt={v.title}
											loading="lazy"
											className="aspect-video w-28 rounded object-cover"
											onError={(e) => {
												(e.currentTarget as HTMLImageElement).style.visibility = 'hidden';
											}}
										/>
									</TableCell>
									<TableCell>
										<div className="font-medium">{v.title}</div>
										<div className="text-xs text-muted-foreground">{v.slug}</div>
									</TableCell>
									<TableCell><CategoryPills categories={v.categories?.length ? v.categories : v.category ? [v.category] : []} /></TableCell>
									<TableCell><StatusBadge value={v.status} /></TableCell>
									<TableCell><StatusBadge value={v.visibility} /></TableCell>
									<TableCell>
										<Badge variant={v.approved_site_count > 0 ? 'success' : 'muted'}>
											{v.approved_site_count} approved
										</Badge>
									</TableCell>
									<TableCell className="text-right tabular-nums">{v.views_count}</TableCell>
									<TableCell className="text-sm text-muted-foreground">{formatDate(v.created_at)}</TableCell>
									<TableCell>
										<DropdownMenu>
											<DropdownMenuTrigger asChild>
												<Button variant="ghost" size="icon" aria-label="Actions">
													<MoreHorizontal className="h-4 w-4" />
												</Button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end">
												<DropdownMenuItem onSelect={() => { setPreviewId(v.id); setPreviewTitle(v.title); setPreviewHLSStatus(v.hls?.status); }}>
													<Eye className="h-4 w-4" /> Preview
												</DropdownMenuItem>
												<DropdownMenuItem asChild>
													<Link href={`/videos/${v.id}`}>
														<Pencil className="h-4 w-4" /> Edit
													</Link>
												</DropdownMenuItem>
												<DropdownMenuItem onSelect={() => copyStreamURL(v)}>
													<Copy className="h-4 w-4" /> Copy stream URL
												</DropdownMenuItem>
												<DropdownMenuItem onSelect={() => toggleVisibility(v)}>
													{v.visibility === 'public' ? 'Make private' : 'Make public'}
												</DropdownMenuItem>
												<DropdownMenuSeparator />
												<DropdownMenuItem onSelect={() => setDistributionTarget({ video: v, action: 'approve' })}>
													<CheckCircle2 className="h-4 w-4" /> Approve Site
												</DropdownMenuItem>
												<DropdownMenuItem onSelect={() => setDistributionTarget({ video: v, action: 'publish' })}>
													<Send className="h-4 w-4" /> Publish To Site
												</DropdownMenuItem>
												<DropdownMenuItem onSelect={() => setDistributionTarget({ video: v, action: 'distribution' })}>
													<Globe2 className="h-4 w-4" /> Site Distribution
												</DropdownMenuItem>
												<DropdownMenuSeparator />
												<DropdownMenuItem destructive onSelect={() => setDeleteTarget(v)}>
													<Trash2 className="h-4 w-4" /> Delete
												</DropdownMenuItem>
											</DropdownMenuContent>
										</DropdownMenu>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			)}

			{pagination && pagination.total_pages > 1 && (
				<div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
					<div>
						Page {pagination.page} of {pagination.total_pages} · {pagination.total} total
					</div>
					<div className="flex gap-2">
						<Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}>
							<ChevronLeft className="h-4 w-4" /> Prev
						</Button>
						<Button variant="outline" size="sm" disabled={page >= pagination.total_pages} onClick={() => setPage((p) => p + 1)}>
							Next <ChevronRight className="h-4 w-4" />
						</Button>
					</div>
				</div>
			)}

			<VideoPreviewModal
				videoId={previewId}
				title={previewTitle}
				hlsStatus={previewHLSStatus}
				onOpenChange={(o) => !o && setPreviewId(null)}
			/>

			<ConfirmDialog
				open={!!deleteTarget}
				onOpenChange={(o) => !o && setDeleteTarget(null)}
				title={`Delete "${deleteTarget?.title}"?`}
				description="The video is soft-deleted on the backend. Files remain on disk until a future cleanup phase removes them."
				confirmLabel="Delete"
				destructive
				pending={deletePending}
				onConfirm={confirmDelete}
			/>

			<SiteDistributionModal
				target={distributionTarget}
				onOpenChange={(open) => !open && setDistributionTarget(null)}
				onSaved={() => {
					mutate(key);
					mutate((k) => typeof k === 'string' && k.startsWith('/api/v1/admin/videos'));
				}}
			/>

			<BulkDistributionModal
				open={bulkOpen}
				videoIds={selectedIds}
				initialStatus={bulkDefaults.status}
				initialSyncMode={bulkDefaults.syncMode}
				onOpenChange={setBulkOpen}
				onSaved={() => {
					setSelectedIds([]);
					mutate(key);
					mutate((k) => typeof k === 'string' && k.startsWith('/api/v1/admin/videos'));
				}}
			/>
		</>
	);
}

type DistributionAction = 'approve' | 'publish' | 'distribution';

function SiteDistributionModal({
	target,
	onOpenChange,
	onSaved,
}: {
	target: { video: Video; action: DistributionAction } | null;
	onOpenChange: (open: boolean) => void;
	onSaved: () => void;
}) {
	const video = target?.video;
	const key = video ? `/api/v1/admin/videos/${video.id}/site-distribution` : null;
	const { data, isLoading, mutate: mutateDistribution } = useSWR<SiteDistributionResponse>(key, () => getSiteDistribution(video!.id));
	const [selectedSiteID, setSelectedSiteID] = useState('');
	const [status, setStatus] = useState<'draft' | 'approved' | 'blocked'>('draft');
	const [syncMode, setSyncMode] = useState<SyncMode>('draft');
	const [pending, setPending] = useState(false);

	const rows = data?.data ?? [];
	const selected = rows.find((row) => row.site_id === selectedSiteID) ?? rows[0];

	useEffect(() => {
		if (!target || rows.length === 0) return;
		const approved = rows.find((row) => row.status === 'approved');
		const first = approved ?? rows[0];
		setSelectedSiteID(first.site_id);
	}, [target, rows]);

	useEffect(() => {
		if (!selected) return;
		if (target?.action === 'publish') {
			setStatus('approved');
			setSyncMode(selected.mapping_sync_mode === 'block' ? 'draft' : 'publish');
			return;
		}
		if (target?.action === 'approve') {
			setStatus('approved');
			setSyncMode(selected.sync_mode === 'block' ? 'draft' : selected.sync_mode);
			return;
		}
		setStatus(selected.status);
		setSyncMode(selected.sync_mode);
	}, [selected, target?.action]);

	async function save() {
		if (!video || !selectedSiteID) return;
		setPending(true);
		try {
			await saveSiteDistribution(video.id, {
				site_id: selectedSiteID,
				status,
				sync_mode: syncMode,
			});
			toast.success(syncMode === 'publish' && status === 'approved' ? 'Video approved for site sync' : 'Site distribution saved');
			await mutateDistribution();
			onSaved();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Distribution update failed');
		} finally {
			setPending(false);
		}
	}

	return (
		<Dialog open={!!target} onOpenChange={onOpenChange}>
			<DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto">
				<DialogHeader>
					<DialogTitle>Site Distribution</DialogTitle>
					<DialogDescription>{video ? video.title : 'Choose how this video is distributed per site.'}</DialogDescription>
				</DialogHeader>

				{isLoading ? (
					<div className="space-y-3">
						<Skeleton className="h-14 w-full" />
						<Skeleton className="h-40 w-full" />
					</div>
				) : rows.length === 0 ? (
					<div className="rounded-md border p-4 text-sm text-muted-foreground">No sites are available.</div>
				) : (
					<div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_280px]">
						<div className="rounded-md border">
							{rows.map((row) => (
								<button
									key={row.site_id}
									type="button"
									onClick={() => setSelectedSiteID(row.site_id)}
									className={`flex w-full items-center justify-between gap-3 border-b p-3 text-left text-sm last:border-b-0 hover:bg-muted ${row.site_id === selectedSiteID ? 'bg-muted' : ''}`}
								>
									<div className="min-w-0">
										<div className="flex flex-wrap items-center gap-2">
											<span className="font-medium">{row.site_name}</span>
											<StatusBadge value={row.status} />
											<Badge variant={row.mapping_status === 'mapped' ? 'success' : row.mapping_status === 'unmapped' ? 'warning' : 'muted'}>
												{row.mapping_status}
											</Badge>
										</div>
										<div className="truncate text-xs text-muted-foreground">{row.site_domain}</div>
									</div>
									<div className="text-right text-xs text-muted-foreground">
										<div>{row.sync_mode}</div>
										{row.remote_category_name && <div className="max-w-36 truncate">{row.remote_category_name}</div>}
									</div>
								</button>
							))}
						</div>

						<div className="space-y-4 rounded-md border p-4">
							{selected && (
								<>
									<div>
										<div className="font-medium">{selected.site_name}</div>
										<div className="text-xs text-muted-foreground">{selected.site_domain}</div>
									</div>

									<div className="space-y-2 rounded-md border p-3 text-sm">
										<div className="font-medium">Video categories</div>
										<CategoryPills categories={video?.categories?.length ? video.categories : video?.category ? [video.category] : []} />
									</div>

									<div className="grid grid-cols-2 gap-3">
										<label className="flex items-center gap-2 rounded-md border p-2 text-sm">
											<input type="checkbox" checked={status === 'approved'} onChange={(e) => setStatus(e.target.checked ? 'approved' : 'draft')} disabled={pending} />
											Approve
										</label>
										<label className="flex items-center gap-2 rounded-md border p-2 text-sm">
											<input type="checkbox" checked={syncMode === 'publish'} onChange={(e) => setSyncMode(e.target.checked ? 'publish' : 'draft')} disabled={pending || selected.mapping_sync_mode === 'block'} />
											Publish
										</label>
									</div>

									<div className="space-y-1.5">
										<Label>Status</Label>
										<Select value={status} onValueChange={(value) => setStatus(value as 'draft' | 'approved' | 'blocked')} disabled={pending}>
											<SelectTrigger><SelectValue /></SelectTrigger>
											<SelectContent>
												<SelectItem value="draft">Draft</SelectItem>
												<SelectItem value="approved">Approved</SelectItem>
												<SelectItem value="blocked">Blocked</SelectItem>
											</SelectContent>
										</Select>
									</div>

									<div className="space-y-1.5">
										<Label>Sync mode</Label>
										<Select value={syncMode} onValueChange={(value) => setSyncMode(value as SyncMode)} disabled={pending}>
											<SelectTrigger><SelectValue /></SelectTrigger>
											<SelectContent>
												<SelectItem value="publish" disabled={selected.mapping_sync_mode === 'block'}>Publish</SelectItem>
												<SelectItem value="draft">Draft</SelectItem>
												<SelectItem value="block">Block</SelectItem>
											</SelectContent>
										</Select>
									</div>

									<div className="space-y-2 text-sm">
										<div className="flex items-center justify-between">
											<span className="text-muted-foreground">Mapping</span>
											<StatusBadge value={selected.mapping_status} />
										</div>
										<div className="flex items-center justify-between">
											<span className="text-muted-foreground">Mapping mode</span>
											<Badge variant={selected.mapping_sync_mode === 'block' ? 'warning' : 'secondary'}>{selected.mapping_sync_mode}</Badge>
										</div>
										<div className="flex items-center justify-between">
											<span className="text-muted-foreground">Remote category</span>
											<span className="max-w-36 truncate">{selected.remote_category_name ?? 'None'}</span>
										</div>
									</div>

									{selected.category_mappings?.length > 0 && (
										<div className="space-y-2 text-sm">
											<div className="font-medium">Category mappings</div>
											<div className="rounded-md border">
												{selected.category_mappings.map((mapping) => (
													<div key={mapping.category_id} className="flex items-center justify-between gap-3 border-b p-2 last:border-b-0">
														<div className="min-w-0">
															<div className="truncate font-medium">{mapping.category_name}</div>
															<div className="truncate text-xs text-muted-foreground">{mapping.remote_category_name ?? 'No remote category'}</div>
														</div>
														<div className="flex shrink-0 items-center gap-2">
															<StatusBadge value={mapping.mapping_status} />
															<Badge variant={mapping.mapping_sync_mode === 'block' ? 'warning' : 'secondary'}>{mapping.mapping_sync_mode}</Badge>
														</div>
													</div>
												))}
											</div>
										</div>
									)}

									{selected.warnings.length > 0 && (
										<div className="rounded-md border border-amber-700 bg-amber-950 p-3 text-xs text-amber-100">
											<div className="mb-1 flex items-center gap-1 font-medium"><ShieldOff className="h-3.5 w-3.5" /> Warning</div>
											{selected.warnings.map((warning) => (
												<div key={warning}>{warning}</div>
											))}
										</div>
									)}
								</>
							)}
						</div>
					</div>
				)}

				<DialogFooter>
					<Button variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>Cancel</Button>
					<Button onClick={save} disabled={pending || !selectedSiteID || rows.length === 0}>
						{pending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
						Save
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

function BulkDistributionModal({
	open,
	videoIds,
	initialStatus,
	initialSyncMode,
	onOpenChange,
	onSaved,
}: {
	open: boolean;
	videoIds: string[];
	initialStatus: 'approved' | 'draft' | 'blocked';
	initialSyncMode: SyncMode;
	onOpenChange: (open: boolean) => void;
	onSaved: () => void;
}) {
	const { data: sitesData } = useSWR<SiteListResponse>(open ? '/api/v1/admin/sites?limit=100&status=active' : null, () => listSites({ limit: 100, status: 'active' }));
	const [siteId, setSiteId] = useState('');
	const [status, setStatus] = useState<'approved' | 'draft' | 'blocked'>('approved');
	const [syncMode, setSyncMode] = useState<SyncMode>('publish');
	const [unmappedBehavior, setUnmappedBehavior] = useState<'skip' | 'draft'>('skip');
	const [pending, setPending] = useState(false);
	const [warnings, setWarnings] = useState<string[]>([]);
	const sites = sitesData?.data ?? [];

	useEffect(() => {
		if (!open) return;
		setStatus(initialStatus);
		setSyncMode(initialSyncMode);
		if (!siteId && sites.length > 0) setSiteId(sites[0].id);
	}, [open, initialStatus, initialSyncMode, siteId, sites]);

	useEffect(() => {
		let cancelled = false;
		async function loadWarnings() {
			if (!open || !siteId || videoIds.length === 0) {
				setWarnings([]);
				return;
			}
			const sample = videoIds.slice(0, 25);
			try {
				const results = await Promise.all(sample.map((id) => getSiteDistribution(id).catch(() => null)));
				if (cancelled) return;
				const next: string[] = [];
				for (const result of results) {
					if (!result) continue;
					const row = result.data.find((r) => r.site_id === siteId);
					if (!row) continue;
					if (row.warnings.length > 0) next.push(`${result.video_id}: ${row.warnings.join(' ')}`);
				}
				setWarnings(next);
			} catch {
				if (!cancelled) setWarnings([]);
			}
		}
		loadWarnings();
		return () => { cancelled = true; };
	}, [open, siteId, videoIds]);

	async function save() {
		if (!siteId || videoIds.length === 0) return;
		setPending(true);
		try {
			const result = await bulkSaveSiteDistribution(siteId, {
				video_ids: videoIds,
				status,
				sync_mode: syncMode,
				unmapped_behavior: unmappedBehavior,
			});
			toast.success(`Bulk distribution saved: ${result.approved} approved, ${result.updated} updated, ${result.skipped} skipped`);
			if (result.errors.length > 0) toast.warning(`${result.errors.length} videos were skipped`);
			onSaved();
			onOpenChange(false);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Bulk distribution failed');
		} finally {
			setPending(false);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-2xl">
				<DialogHeader>
					<DialogTitle>Bulk Video Distribution</DialogTitle>
					<DialogDescription>{videoIds.length} selected videos</DialogDescription>
				</DialogHeader>

				<div className="grid gap-4 sm:grid-cols-2">
					<div className="space-y-1.5 sm:col-span-2">
						<Label>Target Site</Label>
						<Select value={siteId} onValueChange={setSiteId} disabled={pending}>
							<SelectTrigger><SelectValue placeholder="Select site" /></SelectTrigger>
							<SelectContent>
								{sites.map((site) => <SelectItem key={site.id} value={site.id}>{site.name} ({site.domain})</SelectItem>)}
							</SelectContent>
						</Select>
					</div>
					<div className="space-y-1.5">
						<Label>Status</Label>
						<Select value={status} onValueChange={(value) => setStatus(value as 'approved' | 'draft' | 'blocked')} disabled={pending}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								<SelectItem value="approved">Approved</SelectItem>
								<SelectItem value="draft">Draft</SelectItem>
								<SelectItem value="blocked">Blocked</SelectItem>
							</SelectContent>
						</Select>
					</div>
					<div className="space-y-1.5">
						<Label>Sync mode</Label>
						<Select value={syncMode} onValueChange={(value) => setSyncMode(value as SyncMode)} disabled={pending}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								<SelectItem value="publish">Publish</SelectItem>
								<SelectItem value="draft">Draft</SelectItem>
								<SelectItem value="block">Block</SelectItem>
							</SelectContent>
						</Select>
					</div>
					<div className="space-y-1.5 sm:col-span-2">
						<Label>Unmapped/blocked category behavior</Label>
						<Select value={unmappedBehavior} onValueChange={(value) => setUnmappedBehavior(value as 'skip' | 'draft')} disabled={pending}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								<SelectItem value="skip">Skip unmapped</SelectItem>
								<SelectItem value="draft">Approve as draft if unmapped</SelectItem>
							</SelectContent>
						</Select>
					</div>
				</div>

				{warnings.length > 0 && (
					<div className="max-h-40 overflow-auto rounded-md border border-amber-700 bg-amber-950 p-3 text-xs text-amber-100">
						<div className="mb-1 flex items-center gap-1 font-medium"><ShieldOff className="h-3.5 w-3.5" /> Category warnings</div>
						{warnings.map((warning) => <div key={warning}>{warning}</div>)}
					</div>
				)}

				<DialogFooter>
					<Button variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>Cancel</Button>
					<Button onClick={save} disabled={pending || !siteId || videoIds.length === 0}>
						{pending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
						Apply
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

function CategoryPills({ categories }: { categories: Array<{ id: string; name: string }> }) {
	if (categories.length === 0) return <span className="text-sm text-muted-foreground">—</span>;
	return (
		<div className="flex max-w-56 flex-wrap gap-1">
			{categories.map((category) => (
				<Badge key={category.id} variant="secondary">{category.name}</Badge>
			))}
		</div>
	);
}

function TableSkeleton({ rows = 5 }: { rows?: number }) {
	return (
		<div className="space-y-2 rounded-lg border p-2">
			{Array.from({ length: rows }).map((_, i) => (
				<div key={i} className="flex items-center gap-3 p-2">
					<Skeleton className="aspect-video w-28 rounded" />
					<div className="flex-1 space-y-2">
						<Skeleton className="h-4 w-40" />
						<Skeleton className="h-3 w-24" />
					</div>
					<Skeleton className="h-5 w-16" />
					<Skeleton className="h-5 w-16" />
				</div>
			))}
		</div>
	);
}
