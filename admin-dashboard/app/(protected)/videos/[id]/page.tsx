'use client';

import { useParams } from 'next/navigation';
import Link from 'next/link';
import useSWR, { mutate } from 'swr';
import { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { ArrowLeft, Eye, Loader2, Send } from 'lucide-react';
import { PageHeader } from '@/components/page-header';
import { CategoryMultiSelect } from '@/components/category-multi-select';
import { StatusBadge } from '@/components/status-badge';
import { VideoPreviewModal } from '@/components/video-preview-modal';
import { Button } from '@/components/ui/button';
import { Input, Textarea } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { swrFetcher } from '@/lib/api/client';
import { updateVideo } from '@/lib/api/videos';
import { upsertPublication } from '@/lib/api/publications';
import { formatBytes, formatDate, formatDuration } from '@/lib/utils/format';
import type { CategoryListResponse, PublicationListResponse, SiteListResponse, Video } from '@/lib/types';

export default function VideoEditPage() {
	const params = useParams<{ id: string }>();
	const id = params.id;

	const key = `/api/v1/admin/videos/${id}`;
	const { data: video, isLoading, error } = useSWR<Video>(key, swrFetcher);
	const { data: cats } = useSWR<CategoryListResponse>('/api/v1/admin/categories?limit=100&status=active', swrFetcher);
	const { data: sites } = useSWR<SiteListResponse>('/api/v1/admin/sites?limit=100&status=active', swrFetcher);
	const { data: publications } = useSWR<PublicationListResponse>(video ? `/api/v1/admin/videos/${video.id}/publications?limit=100` : null, swrFetcher);

	const [title, setTitle] = useState('');
	const [slug, setSlug] = useState('');
	const [description, setDescription] = useState('');
	const [status, setStatus] = useState<string>('');
	const [visibility, setVisibility] = useState<string>('');
	const [categoryIds, setCategoryIds] = useState<string[]>([]);
	const [thumbSecond, setThumbSecond] = useState<string>('');
	const [pending, setPending] = useState(false);
	const [preview, setPreview] = useState(false);

	useEffect(() => {
		if (!video) return;
		setTitle(video.title);
		setSlug(video.slug);
		setDescription(video.description ?? '');
		setStatus(video.status);
		setVisibility(video.visibility);
		setCategoryIds(video.category_ids?.length ? video.category_ids : [video.category.id]);
		setThumbSecond(String(video.thumbnail.second));
	}, [video]);

	if (isLoading) {
		return (
			<>
				<div className="mb-2"><BackButton /></div>
				<Skeleton className="mb-6 h-8 w-64" />
				<div className="grid gap-6 lg:grid-cols-3">
					<Skeleton className="lg:col-span-2 aspect-video" />
					<Skeleton className="h-96" />
				</div>
			</>
		);
	}

	if (error || !video) {
		return (
			<>
				<div className="mb-2"><BackButton /></div>
				<div className="rounded-lg border border-destructive bg-red-950 p-6 text-sm text-red-100">
					Could not load this video.
				</div>
			</>
		);
	}

	async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		if (!video) return;
		if (categoryIds.length === 0) {
			toast.error('Select at least one category.');
			return;
		}
		setPending(true);
		try {
			// Build a "changed only" payload — empty strings are no-ops on the backend.
			const payload: Record<string, string | number | string[] | undefined> = {};
			if (title !== video.title) payload.title = title.trim();
			if (slug !== video.slug) payload.slug = slug.trim();
			if (description !== (video.description ?? '')) payload.description = description.trim();
			if (status !== video.status) payload.status = status;
			if (visibility !== video.visibility) payload.visibility = visibility;
			const originalCategoryIds = video.category_ids?.length ? video.category_ids : [video.category.id];
			if (categoryIds.join('|') !== originalCategoryIds.join('|')) {
				payload.category_id = categoryIds[0];
				payload.category_ids = categoryIds;
			}
			if (Number(thumbSecond) !== video.thumbnail.second) payload.thumbnail_second = Number(thumbSecond);

			if (Object.keys(payload).length === 0) {
				toast.info('No changes to save.');
				setPending(false);
				return;
			}

			await updateVideo(video.id, payload as never);
			toast.success('Saved');
			mutate(key);
			mutate((k) => typeof k === 'string' && k.startsWith('/api/v1/admin/videos'));
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Save failed');
		} finally {
			setPending(false);
		}
	}

	return (
		<>
			<div className="mb-2"><BackButton /></div>
			<PageHeader
				title={video.title}
				description={`Uploaded ${formatDate(video.created_at)} · ${formatBytes(video.video.size_bytes)} · ${formatDuration(video.duration_seconds)}`}
				actions={
					<>
						<Button variant="outline" onClick={() => setPreview(true)}>
							<Eye className="h-4 w-4" /> Preview
						</Button>
					</>
				}
			/>

			<div className="grid gap-6 lg:grid-cols-3">
				<div className="space-y-4 lg:col-span-2">
					<Card>
						<CardContent className="p-0">
							{/* eslint-disable-next-line @next/next/no-img-element */}
							<img src={video.thumbnail_url} alt={video.title} className="aspect-video w-full rounded-t-lg object-cover" />
							<div className="grid grid-cols-2 gap-4 p-4 text-sm sm:grid-cols-5">
								<MetaItem label="Status" value={<StatusBadge value={video.status} />} />
								<MetaItem label="Visibility" value={<StatusBadge value={video.visibility} />} />
								<MetaItem label="Views" value={String(video.views_count)} />
								<MetaItem label="Duration" value={formatDuration(video.duration_seconds)} />
								<MetaItem
									label="HLS"
									value={
										<div className="flex flex-col gap-1">
											<StatusBadge value={video.hls?.status ?? 'pending'} />
											{video.hls?.status === 'ready' && video.hls.variants && video.hls.variants.length > 0 && (
												<div className="text-xs text-muted-foreground">
													Qualities: {video.hls.variants.join(', ')}
												</div>
											)}
										</div>
									}
								/>
							</div>
							{video.hls?.status === 'failed' && video.hls.error && (
								<div className="border-t bg-red-950 px-4 py-3 text-xs text-red-100">
									HLS conversion failed: {video.hls.error}
								</div>
							)}
						</CardContent>
					</Card>

					<Card>
						<CardHeader><CardTitle>File</CardTitle></CardHeader>
						<CardContent className="grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
							<MetaItem label="Original name" value={video.video.original_name} />
							<MetaItem label="MIME" value={video.video.mime_type} />
							<MetaItem label="Size" value={formatBytes(video.video.size_bytes)} />
							<MetaItem label="Stored filename" value={<code className="text-xs">{video.video.filename}</code>} />
						</CardContent>
					</Card>

					<PublishingPanel
						video={video}
						sites={sites?.data ?? []}
						publications={publications?.data ?? []}
					/>
				</div>

				<form onSubmit={handleSubmit} className="space-y-4">
					<Card>
						<CardHeader><CardTitle>Metadata</CardTitle></CardHeader>
						<CardContent className="space-y-4">
							<div className="space-y-1.5">
								<Label htmlFor="title">Title</Label>
								<Input id="title" value={title} onChange={(e) => setTitle(e.target.value)} disabled={pending} />
							</div>
							<div className="space-y-1.5">
								<Label htmlFor="slug">Slug</Label>
								<Input id="slug" value={slug} onChange={(e) => setSlug(e.target.value)} disabled={pending} />
							</div>
							<div className="space-y-1.5">
								<Label htmlFor="description">Description</Label>
								<Textarea id="description" value={description} onChange={(e) => setDescription(e.target.value)} disabled={pending} />
							</div>
							<div className="space-y-1.5">
								<Label>Categories</Label>
								<CategoryMultiSelect
									categories={cats?.data ?? []}
									value={categoryIds}
									onChange={setCategoryIds}
									disabled={pending}
								/>
							</div>
							<div className="space-y-1.5">
								<Label>Status</Label>
								<Select value={status} onValueChange={setStatus} disabled={pending}>
									<SelectTrigger><SelectValue /></SelectTrigger>
									<SelectContent>
										<SelectItem value="ready">Ready</SelectItem>
										<SelectItem value="draft">Draft</SelectItem>
										<SelectItem value="processing">Processing</SelectItem>
										<SelectItem value="blocked">Blocked</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div className="space-y-1.5">
								<Label>Visibility</Label>
								<Select value={visibility} onValueChange={setVisibility} disabled={pending}>
									<SelectTrigger><SelectValue /></SelectTrigger>
									<SelectContent>
										<SelectItem value="public">Public</SelectItem>
										<SelectItem value="private">Private</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div className="space-y-1.5">
								<Label htmlFor="thumb-sec">Thumbnail second</Label>
								<Input
									id="thumb-sec"
									type="number"
									min={0}
									max={video.duration_seconds && video.duration_seconds > 0 ? video.duration_seconds - 1 : 86400}
									value={thumbSecond}
									onChange={(e) => setThumbSecond(e.target.value)}
									disabled={pending}
								/>
								<p className="text-xs text-muted-foreground">
									{video.duration_seconds && video.duration_seconds > 0
										? `Choose a second between 0 and ${video.duration_seconds - 1} for this video (duration ${formatDuration(video.duration_seconds)}).`
										: 'Duration unknown for this video — the backend will auto-correct to the nearest valid second.'}
									{' '}If you pick a value past the end, the backend clamps it automatically and shows the corrected second after save.
								</p>
								{(() => {
									const dur = video.duration_seconds;
									const n = Number(thumbSecond);
									if (dur && dur > 0 && Number.isFinite(n) && n >= dur) {
										return (
											<p className="text-xs text-amber-600 dark:text-amber-400">
												{`Past the end of the video — will be clamped to ${dur - 1} on save.`}
											</p>
										);
									}
									return null;
								})()}
							</div>
						</CardContent>
					</Card>

					<Button type="submit" className="w-full" disabled={pending}>
						{pending ? (<><Loader2 className="h-4 w-4 animate-spin" /> Saving…</>) : 'Save changes'}
					</Button>
				</form>
			</div>

			<VideoPreviewModal
				videoId={preview ? video.id : null}
				title={video.title}
				hlsStatus={video.hls?.status}
				onOpenChange={(o) => setPreview(o)}
			/>
		</>
	);
}

function PublishingPanel({ video, sites, publications }: { video: Video; sites: { id: string; name: string; domain: string }[]; publications: NonNullable<PublicationListResponse['data']> }) {
	const [siteId, setSiteId] = useState('');
	const [status, setStatus] = useState<'approved' | 'disabled'>('approved');
	const [categoryMapping, setCategoryMapping] = useState('');
	const [tags, setTags] = useState('');
	const [notes, setNotes] = useState('');
	const [pending, setPending] = useState(false);

	async function submit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		if (!siteId) return;
		setPending(true);
		try {
			await upsertPublication(siteId, video.id, {
				status,
				category_mapping: categoryMapping.trim() || null,
				tags: tags.split(',').map((t) => t.trim()).filter(Boolean),
				notes: notes.trim() || null,
			});
			toast.success(status === 'approved' ? 'Video approved for site' : 'Video disabled for site');
			mutate(`/api/v1/admin/videos/${video.id}/publications?limit=100`);
			mutate((k) => typeof k === 'string' && k.includes('/publications'));
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Publishing update failed');
		} finally {
			setPending(false);
		}
	}

	return (
		<Card>
			<CardHeader><CardTitle>Site publishing</CardTitle></CardHeader>
			<CardContent className="space-y-4">
				<form onSubmit={submit} className="grid gap-3 sm:grid-cols-2">
					<div className="space-y-1.5">
						<Label>Target site</Label>
						<Select value={siteId} onValueChange={setSiteId} disabled={pending}>
							<SelectTrigger><SelectValue placeholder="Choose site" /></SelectTrigger>
							<SelectContent>
								{sites.map((s) => (
									<SelectItem key={s.id} value={s.id}>{s.name} · {s.domain}</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>
					<div className="space-y-1.5">
						<Label>Status</Label>
						<Select value={status} onValueChange={(v) => setStatus(v as 'approved' | 'disabled')} disabled={pending}>
							<SelectTrigger><SelectValue /></SelectTrigger>
							<SelectContent>
								<SelectItem value="approved">Approved</SelectItem>
								<SelectItem value="disabled">Disabled</SelectItem>
							</SelectContent>
						</Select>
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="category-mapping">Category mapping</Label>
						<Input id="category-mapping" value={categoryMapping} onChange={(e) => setCategoryMapping(e.target.value)} placeholder={(video.categories ?? [video.category]).map((c) => c.name).join(', ')} disabled={pending} />
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="publication-tags">Tags</Label>
						<Input id="publication-tags" value={tags} onChange={(e) => setTags(e.target.value)} placeholder="Comma separated" disabled={pending} />
					</div>
					<div className="space-y-1.5 sm:col-span-2">
						<Label htmlFor="publication-notes">Notes</Label>
						<Textarea id="publication-notes" value={notes} onChange={(e) => setNotes(e.target.value)} disabled={pending} />
					</div>
					<div className="sm:col-span-2">
						<Button type="submit" disabled={pending || !siteId}>
							<Send className="h-4 w-4" /> {pending ? 'Saving…' : 'Save publishing'}
						</Button>
					</div>
				</form>

				<div className="rounded-md border">
					{publications.length === 0 ? (
						<div className="p-4 text-sm text-muted-foreground">Not approved for any site.</div>
					) : (
						<div className="divide-y">
							{publications.map((p) => {
								const site = sites.find((s) => s.id === p.site_id);
								return (
									<div key={p.id} className="flex items-center justify-between gap-3 p-3 text-sm">
										<div>
											<div className="font-medium">{site?.name ?? p.site_id}</div>
											<div className="text-xs text-muted-foreground">{site?.domain ?? 'Unknown site'} · updated {formatDate(p.updated_at)}</div>
										</div>
										<StatusBadge value={p.status} />
									</div>
								);
							})}
						</div>
					)}
				</div>
			</CardContent>
		</Card>
	);
}

function BackButton() {
	return (
		<Button asChild variant="ghost" size="sm">
			<Link href="/videos"><ArrowLeft className="h-4 w-4" /> Back to videos</Link>
		</Button>
	);
}

function MetaItem({ label, value }: { label: string; value: React.ReactNode }) {
	return (
		<div>
			<div className="text-xs text-muted-foreground">{label}</div>
			<div className="mt-1 font-medium">{value}</div>
		</div>
	);
}
