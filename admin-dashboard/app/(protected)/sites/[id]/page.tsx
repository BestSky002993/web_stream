'use client';

import Link from 'next/link';
import { useParams } from 'next/navigation';
import { useEffect, useMemo, useState } from 'react';
import useSWR, { mutate } from 'swr';
import { toast } from 'sonner';
import { ArrowLeft, Check, ChevronLeft, ChevronRight, RefreshCw, Trash2, PowerOff, Wand2, Save, Ban, Send, Loader2, ShieldOff } from 'lucide-react';
import { PageHeader } from '@/components/page-header';
import { StatusBadge } from '@/components/status-badge';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { swrFetcher } from '@/lib/api/client';
import { approvePublication, disablePublication, removePublication, resyncPublication } from '@/lib/api/publications';
import { bulkSaveSiteDistribution, getSiteDistribution, listVideos } from '@/lib/api/videos';
import { listCategoryMappings, listRemoteCategories, saveCategoryMappings, suggestCategoryMappings } from '@/lib/api/category-mappings';
import { formatDate } from '@/lib/utils/format';
import type { CategoryListResponse, CategoryMappingListResponse, CategoryMappingRow, PublicationListResponse, RemoteCategoryListResponse, Site, SyncMode, VideoListResponse } from '@/lib/types';

const NONE = '__none__';
const SITE_BULK_PAGE_SIZE = 20;

export default function SiteDetailPage() {
	const params = useParams<{ id: string }>();
	const siteId = params.id;
	const [tab, setTab] = useState<'videos' | 'mapping'>('videos');
	const { data: site, isLoading: siteLoading } = useSWR<Site>(`/api/v1/admin/sites/${siteId}`, swrFetcher);
	const publicationKey = `/api/v1/admin/sites/${siteId}/publications?limit=100`;
	const { data, isLoading } = useSWR<PublicationListResponse>(publicationKey, swrFetcher);
	const items = data?.data ?? [];
	const [bulkOpen, setBulkOpen] = useState(false);

	function refreshPublications() {
		mutate(publicationKey);
		mutate((k) => typeof k === 'string' && k.includes('/publications'));
	}

	async function run(action: () => Promise<unknown>, label: string) {
		try {
			await action();
			toast.success(label);
			refreshPublications();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Action failed');
		}
	}

	if (siteLoading) return <Skeleton className="h-48" />;

	return (
		<>
			<div className="mb-2">
				<Button asChild variant="ghost" size="sm">
					<Link href="/sites"><ArrowLeft className="h-4 w-4" /> Back to sites</Link>
				</Button>
			</div>
			<PageHeader
				title={site?.name ?? 'Site'}
				description={`${site?.domain ?? siteId} · controlled WordPress sync contract`}
			/>

			<div className="mb-4 flex gap-2 border-b">
				<Button variant={tab === 'videos' ? 'default' : 'ghost'} size="sm" onClick={() => setTab('videos')}>Approved Videos</Button>
				<Button variant={tab === 'mapping' ? 'default' : 'ghost'} size="sm" onClick={() => setTab('mapping')}>Category Mapping</Button>
			</div>

			{tab === 'videos' ? (
				<>
					<div className="mb-3 flex justify-end">
						<Button size="sm" onClick={() => setBulkOpen(true)}><Send className="h-4 w-4" /> Bulk approve videos</Button>
					</div>
					<ApprovedVideosTable items={items} isLoading={isLoading} run={run} />
				</>
			) : (
				<CategoryMappingPanel siteId={siteId} publications={items} />
			)}
			<SiteBulkApproveModal
				open={bulkOpen}
				siteId={siteId}
				onOpenChange={setBulkOpen}
				onSaved={refreshPublications}
			/>
		</>
	);
}

function ApprovedVideosTable({ items, isLoading, run }: { items: NonNullable<PublicationListResponse['data']>; isLoading: boolean; run: (action: () => Promise<unknown>, label: string) => Promise<void> }) {
	if (isLoading) return <Skeleton className="h-64" />;
	return (
		<div className="rounded-lg border">
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead>Video</TableHead>
						<TableHead>Category</TableHead>
						<TableHead>Status</TableHead>
						<TableHead>Remote post</TableHead>
						<TableHead>Last synced</TableHead>
						<TableHead className="w-56 text-right">Actions</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{items.length === 0 ? (
						<TableRow>
							<TableCell colSpan={6} className="h-24 text-center text-sm text-muted-foreground">
								No approved videos yet. Approve videos from a video detail page.
							</TableCell>
						</TableRow>
					) : items.map((p) => (
						<TableRow key={p.id}>
							<TableCell>
								<div className="font-medium">{p.video?.title ?? p.video_id}</div>
								<div className="text-xs text-muted-foreground">{p.video?.slug}</div>
							</TableCell>
							<TableCell><CategoryPills categories={p.video?.categories?.length ? p.video.categories : p.video?.category ? [p.video.category] : []} /></TableCell>
							<TableCell><StatusBadge value={p.status} /></TableCell>
							<TableCell className="text-sm text-muted-foreground">{p.remote_post_id ?? '—'}</TableCell>
							<TableCell className="text-sm text-muted-foreground">{p.last_synced_at ? formatDate(p.last_synced_at) : 'Never'}</TableCell>
							<TableCell>
								<div className="flex justify-end gap-1">
									<Button size="icon" variant="ghost" title="Approve" onClick={() => run(() => approvePublication(p.id), 'Publication approved')}><Check className="h-4 w-4" /></Button>
									<Button size="icon" variant="ghost" title="Disable" onClick={() => run(() => disablePublication(p.id), 'Publication disabled')}><PowerOff className="h-4 w-4" /></Button>
									<Button size="icon" variant="ghost" title="Resync" onClick={() => run(() => resyncPublication(p.id), 'Publication marked for resync')}><RefreshCw className="h-4 w-4" /></Button>
									<Button size="icon" variant="ghost" title="Remove" onClick={() => run(() => removePublication(p.id), 'Publication removed')}><Trash2 className="h-4 w-4" /></Button>
								</div>
							</TableCell>
						</TableRow>
					))}
				</TableBody>
			</Table>
		</div>
	);
}

function SiteBulkApproveModal({
	open,
	siteId,
	onOpenChange,
	onSaved,
}: {
	open: boolean;
	siteId: string;
	onOpenChange: (open: boolean) => void;
	onSaved: () => void;
}) {
	const [page, setPage] = useState(1);
	const [categoryId, setCategoryId] = useState('');
	const [visibility, setVisibility] = useState('');
	const [statusFilter, setStatusFilter] = useState('');
	const [onlyNotApproved, setOnlyNotApproved] = useState(true);
	const [selectedIds, setSelectedIds] = useState<string[]>([]);
	const [distributionStatus, setDistributionStatus] = useState<'approved' | 'draft' | 'blocked'>('approved');
	const [syncMode, setSyncMode] = useState<SyncMode>('publish');
	const [unmappedBehavior, setUnmappedBehavior] = useState<'skip' | 'draft'>('skip');
	const [pending, setPending] = useState(false);
	const [selectingAll, setSelectingAll] = useState(false);
	const [warnings, setWarnings] = useState<string[]>([]);

	const query = {
		page,
		limit: SITE_BULK_PAGE_SIZE,
		sort: '-created_at',
		category_id: categoryId,
		visibility,
		status: statusFilter,
		not_approved_site_id: onlyNotApproved ? siteId : '',
	};
	const videosKey = open ? `/site-bulk/${siteId}/${new URLSearchParams(Object.entries(query).filter(([, v]) => v !== '').map(([k, v]) => [k, String(v)])).toString()}` : null;
	const { data: videosData, isLoading } = useSWR<VideoListResponse>(videosKey, () => listVideos(query));
	const { data: cats } = useSWR<CategoryListResponse>(open ? '/api/v1/admin/categories?limit=100' : null, swrFetcher);
	const items = videosData?.data ?? [];
	const pagination = videosData?.pagination;
	const selectedOnPage = items.filter((v) => selectedIds.includes(v.id)).length;
	const allPageSelected = items.length > 0 && selectedOnPage === items.length;
	const pageNumbers = Array.from({ length: Math.min(5, pagination?.total_pages ?? 0) }, (_, i) => i + 1);

	useEffect(() => {
		if (!open) {
			setSelectedIds([]);
			setWarnings([]);
			setPage(1);
		}
	}, [open]);

	useEffect(() => {
		setPage(1);
		setSelectedIds([]);
	}, [categoryId, visibility, statusFilter, onlyNotApproved]);

	useEffect(() => {
		let cancelled = false;
		async function loadWarnings() {
			if (!open || selectedIds.length === 0) {
				setWarnings([]);
				return;
			}
			const sample = selectedIds.slice(0, 25);
			const results = await Promise.all(sample.map((id) => getSiteDistribution(id).catch(() => null)));
			if (cancelled) return;
			const next: string[] = [];
			for (const result of results) {
				if (!result) continue;
				const row = result.data.find((r) => r.site_id === siteId);
				if (!row || row.warnings.length === 0) continue;
				next.push(`${result.video_id}: ${row.warnings.join(' ')}`);
			}
			setWarnings(next);
		}
		loadWarnings();
		return () => { cancelled = true; };
	}, [open, selectedIds, siteId]);

	function toggleSelected(id: string, checked: boolean) {
		setSelectedIds((current) => checked ? Array.from(new Set([...current, id])) : current.filter((v) => v !== id));
	}

	function togglePage(checked: boolean) {
		const pageIds = items.map((v) => v.id);
		setSelectedIds((current) => checked ? Array.from(new Set([...current, ...pageIds])) : current.filter((id) => !pageIds.includes(id)));
	}

	async function selectAllMatching() {
		setSelectingAll(true);
		try {
			const first = await listVideos({ ...query, page: 1, limit: 100 });
			const ids = new Set(first.data.map((v) => v.id));
			for (let p = 2; p <= first.pagination.total_pages; p += 1) {
				const next = await listVideos({ ...query, page: p, limit: 100 });
				next.data.forEach((v) => ids.add(v.id));
			}
			setSelectedIds(Array.from(ids));
			toast.success(`${ids.size} matching videos selected`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Could not select all matching videos');
		} finally {
			setSelectingAll(false);
		}
	}

	async function save() {
		if (selectedIds.length === 0) return;
		setPending(true);
		try {
			const result = await bulkSaveSiteDistribution(siteId, {
				video_ids: selectedIds,
				status: distributionStatus,
				sync_mode: syncMode,
				unmapped_behavior: unmappedBehavior,
			});
			toast.success(`Bulk distribution saved: ${result.approved} approved, ${result.updated} updated, ${result.skipped} skipped`);
			if (result.errors.length > 0) toast.warning(`${result.errors.length} videos were skipped`);
			onSaved();
			onOpenChange(false);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Bulk approval failed');
		} finally {
			setPending(false);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-h-[90vh] max-w-5xl overflow-y-auto">
				<DialogHeader>
					<DialogTitle>Bulk Approve Videos</DialogTitle>
					<DialogDescription>{selectedIds.length} selected videos</DialogDescription>
				</DialogHeader>

				<div className="grid gap-3 md:grid-cols-4">
					<div className="space-y-1.5">
						<Label>Category</Label>
						<Select value={categoryId || 'all'} onValueChange={(value) => setCategoryId(value === 'all' ? '' : value)} disabled={pending}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								<SelectItem value="all">All categories</SelectItem>
								{cats?.data.map((c) => <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>)}
							</SelectContent>
						</Select>
					</div>
					<div className="space-y-1.5">
						<Label>Visibility</Label>
						<Select value={visibility || 'all'} onValueChange={(value) => setVisibility(value === 'all' ? '' : value)} disabled={pending}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								<SelectItem value="all">All visibility</SelectItem>
								<SelectItem value="public">Public</SelectItem>
								<SelectItem value="private">Private</SelectItem>
							</SelectContent>
						</Select>
					</div>
					<div className="space-y-1.5">
						<Label>Status</Label>
						<Select value={statusFilter || 'all'} onValueChange={(value) => setStatusFilter(value === 'all' ? '' : value)} disabled={pending}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								<SelectItem value="all">All statuses</SelectItem>
								<SelectItem value="ready">Ready</SelectItem>
								<SelectItem value="draft">Draft</SelectItem>
								<SelectItem value="processing">Processing</SelectItem>
								<SelectItem value="blocked">Blocked</SelectItem>
							</SelectContent>
						</Select>
					</div>
					<label className="mt-7 flex h-10 items-center gap-2 rounded-md border px-3 text-sm">
						<input type="checkbox" checked={onlyNotApproved} onChange={(e) => setOnlyNotApproved(e.currentTarget.checked)} disabled={pending} />
						Not yet approved
					</label>
				</div>

				<div className="grid gap-3 md:grid-cols-3">
					<div className="space-y-1.5">
						<Label>Distribution status</Label>
						<Select value={distributionStatus} onValueChange={(value) => setDistributionStatus(value as 'approved' | 'draft' | 'blocked')} disabled={pending}>
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
					<div className="space-y-1.5">
						<Label>Unmapped behavior</Label>
						<Select value={unmappedBehavior} onValueChange={(value) => setUnmappedBehavior(value as 'skip' | 'draft')} disabled={pending}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								<SelectItem value="skip">Skip unmapped</SelectItem>
								<SelectItem value="draft">Approve as draft if unmapped</SelectItem>
							</SelectContent>
						</Select>
					</div>
				</div>

				<div className="flex flex-wrap items-center justify-between gap-2">
					<div className="flex gap-2">
						<Button size="sm" variant="outline" onClick={() => togglePage(!allPageSelected)} disabled={items.length === 0 || pending}>
							{allPageSelected ? 'Clear current page' : 'Select current page'}
						</Button>
						<Button size="sm" variant="outline" onClick={selectAllMatching} disabled={selectingAll || pending}>
							{selectingAll ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
							Select all matching filter
						</Button>
					</div>
					<div className="text-sm text-muted-foreground">{selectedIds.length} selected</div>
				</div>

				<div className="rounded-lg border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead className="w-10">
									<input type="checkbox" aria-label="Select current page" checked={allPageSelected} onChange={(e) => togglePage(e.currentTarget.checked)} />
								</TableHead>
								<TableHead>Video</TableHead>
								<TableHead>Category</TableHead>
								<TableHead>Status</TableHead>
								<TableHead>Visibility</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{isLoading ? (
								<TableRow><TableCell colSpan={5} className="h-24"><Skeleton className="h-12 w-full" /></TableCell></TableRow>
							) : items.length === 0 ? (
								<TableRow><TableCell colSpan={5} className="h-24 text-center text-sm text-muted-foreground">No videos match these filters.</TableCell></TableRow>
							) : items.map((v) => (
								<TableRow key={v.id}>
									<TableCell><input type="checkbox" aria-label={`Select ${v.title}`} checked={selectedIds.includes(v.id)} onChange={(e) => toggleSelected(v.id, e.currentTarget.checked)} /></TableCell>
									<TableCell>
										<div className="font-medium">{v.title}</div>
										<div className="text-xs text-muted-foreground">{v.slug}</div>
									</TableCell>
									<TableCell><CategoryPills categories={v.categories?.length ? v.categories : v.category ? [v.category] : []} /></TableCell>
									<TableCell><StatusBadge value={v.status} /></TableCell>
									<TableCell><StatusBadge value={v.visibility} /></TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>

				{pagination && pagination.total_pages > 1 && (
					<div className="flex flex-wrap items-center justify-between gap-2 text-sm text-muted-foreground">
						<div>Page {pagination.page} of {pagination.total_pages} · {pagination.total} matching</div>
						<div className="flex gap-1">
							<Button variant="outline" size="icon" disabled={page <= 1} onClick={() => setPage((p) => Math.max(1, p - 1))}><ChevronLeft className="h-4 w-4" /></Button>
							{pageNumbers.map((n) => (
								<Button key={n} variant={page === n ? 'default' : 'outline'} size="sm" onClick={() => setPage(n)}>{n}</Button>
							))}
							<Button variant="outline" size="icon" disabled={page >= pagination.total_pages} onClick={() => setPage((p) => p + 1)}><ChevronRight className="h-4 w-4" /></Button>
						</div>
					</div>
				)}

				{warnings.length > 0 && (
					<div className="max-h-36 overflow-auto rounded-md border border-amber-700 bg-amber-950 p-3 text-xs text-amber-100">
						<div className="mb-1 flex items-center gap-1 font-medium"><ShieldOff className="h-3.5 w-3.5" /> Category warnings</div>
						{warnings.map((warning) => <div key={warning}>{warning}</div>)}
					</div>
				)}

				<DialogFooter>
					<Button variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>Cancel</Button>
					<Button onClick={save} disabled={pending || selectedIds.length === 0}>
						{pending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
						Apply
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}

function CategoryMappingPanel({ siteId, publications }: { siteId: string; publications: NonNullable<PublicationListResponse['data']> }) {
	const remoteKey = `/api/v1/admin/sites/${siteId}/remote-categories`;
	const mappingKey = `/api/v1/admin/sites/${siteId}/category-mappings`;
	const { data: remoteData, isLoading: remoteLoading } = useSWR<RemoteCategoryListResponse>(remoteKey, () => listRemoteCategories(siteId));
	const { data: mappingData, isLoading: mappingLoading } = useSWR<CategoryMappingListResponse>(mappingKey, () => listCategoryMappings(siteId));
	const [drafts, setDrafts] = useState<Record<string, CategoryMappingRow>>({});
	const rows = mappingData?.data ?? [];
	const remotes = remoteData?.data ?? [];
	const effectiveRows = rows.map((row) => drafts[row.backend_category.id] ?? row);
	const approvedCategories = useMemo(() => {
		const ids = new Set<string>();
		for (const p of publications) {
			if (p.status !== 'approved' && p.status !== 'synced') continue;
			const categories = p.video?.categories?.length ? p.video.categories : p.video?.category ? [p.video.category] : [];
			for (const category of categories) ids.add(category.id);
		}
		return ids;
	}, [publications]);
	const warnings = effectiveRows.filter((row) => approvedCategories.has(row.backend_category.id) && (!row.enabled || !row.remote_category_id || row.sync_mode === 'block'));

	function patch(row: CategoryMappingRow, changes: Partial<CategoryMappingRow>) {
		setDrafts((current) => ({ ...current, [row.backend_category.id]: { ...row, ...changes } }));
	}

	async function save(rowsToSave: CategoryMappingRow[], label = 'Category mapping saved') {
		try {
			await saveCategoryMappings(siteId, rowsToSave.map((row) => ({
				backend_category_id: row.backend_category.id,
				remote_category_id: row.remote_category_id,
				enabled: row.enabled,
				sync_mode: row.sync_mode,
				confirmed_by_stream: row.confirmed_by_stream,
			})));
			setDrafts({});
			mutate(mappingKey);
			toast.success(label);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Save failed');
		}
	}

	async function suggest() {
		try {
			await suggestCategoryMappings(siteId);
			setDrafts({});
			mutate(mappingKey);
			toast.success('Same-name suggestions created disabled by default');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Suggestion failed');
		}
	}

	function bulkDisableUnmapped() {
		const next: Record<string, CategoryMappingRow> = {};
		for (const row of effectiveRows) {
			if (!row.remote_category_id) next[row.backend_category.id] = { ...row, enabled: false, sync_mode: 'block', confirmed_by_stream: false };
		}
		setDrafts((current) => ({ ...current, ...next }));
	}

	if (remoteLoading || mappingLoading) return <Skeleton className="h-64" />;

	return (
		<div className="space-y-4">
			<div className="flex flex-wrap items-center justify-between gap-2">
				<div className="flex gap-2">
					<Button size="sm" variant="outline" onClick={suggest}><Wand2 className="h-4 w-4" /> Suggest same-name</Button>
					<Button size="sm" variant="outline" onClick={bulkDisableUnmapped}><Ban className="h-4 w-4" /> Bulk disable unmapped</Button>
				</div>
				<Button size="sm" onClick={() => save(Object.values(drafts), 'All mapping changes saved')} disabled={Object.keys(drafts).length === 0}><Save className="h-4 w-4" /> Save changes</Button>
			</div>

			{remotes.length === 0 && (
				<div className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
					No WordPress categories imported yet. Ask the WordPress admin to push categories from the plugin.
				</div>
			)}
			{warnings.length > 0 && (
				<div className="rounded-md border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
					This site has approved videos whose category is not mapped for this site.
				</div>
			)}

			<div className="rounded-lg border">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Backend Category</TableHead>
							<TableHead>Remote WP Category</TableHead>
							<TableHead>Enabled</TableHead>
							<TableHead>Sync Mode</TableHead>
							<TableHead>WP Confirmed</TableHead>
							<TableHead>Stream Confirmed</TableHead>
							<TableHead className="w-24 text-right">Actions</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{effectiveRows.map((row) => (
							<TableRow key={row.backend_category.id}>
								<TableCell>
									<div className="font-medium">{row.backend_category.name}</div>
									<div className="text-xs text-muted-foreground">{row.backend_category.slug}</div>
								</TableCell>
								<TableCell>
									<Select value={row.remote_category_id ?? NONE} onValueChange={(value) => patch(row, { remote_category_id: value === NONE ? null : value, enabled: value === NONE ? false : row.enabled })}>
										<SelectTrigger className="min-w-52"><SelectValue /></SelectTrigger>
										<SelectContent>
											<SelectItem value={NONE}>No mapping</SelectItem>
											{remotes.map((remote) => <SelectItem key={remote.id} value={remote.id}>{remote.remote_name} ({remote.remote_slug})</SelectItem>)}
										</SelectContent>
									</Select>
								</TableCell>
								<TableCell><Button size="sm" variant={row.enabled ? 'default' : 'outline'} onClick={() => patch(row, { enabled: !row.enabled })}>{row.enabled ? 'Enabled' : 'Disabled'}</Button></TableCell>
								<TableCell>
									<Select value={row.sync_mode} onValueChange={(value) => patch(row, { sync_mode: value as SyncMode })}>
										<SelectTrigger className="w-28"><SelectValue /></SelectTrigger>
										<SelectContent>
											<SelectItem value="block">Block</SelectItem>
											<SelectItem value="draft">Draft</SelectItem>
											<SelectItem value="publish">Publish</SelectItem>
										</SelectContent>
									</Select>
								</TableCell>
								<TableCell><Badge variant={row.confirmed_by_wp ? 'success' : 'muted'}>{row.confirmed_by_wp ? 'Yes' : 'No'}</Badge></TableCell>
								<TableCell><Button size="sm" variant={row.confirmed_by_stream ? 'default' : 'outline'} onClick={() => patch(row, { confirmed_by_stream: !row.confirmed_by_stream })}>{row.confirmed_by_stream ? 'Yes' : 'No'}</Button></TableCell>
								<TableCell className="text-right"><Button size="icon" variant="ghost" title="Save row" onClick={() => save([row])}><Save className="h-4 w-4" /></Button></TableCell>
							</TableRow>
						))}
					</TableBody>
				</Table>
			</div>
		</div>
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
