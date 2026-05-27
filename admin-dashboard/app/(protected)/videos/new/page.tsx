'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import useSWR from 'swr';
import { toast } from 'sonner';
import { ArrowLeft, Loader2 } from 'lucide-react';
import Link from 'next/link';
import { PageHeader } from '@/components/page-header';
import { UploadDropzone } from '@/components/upload-dropzone';
import { CategoryMultiSelect } from '@/components/category-multi-select';
import { Button } from '@/components/ui/button';
import { Input, Textarea } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Card, CardContent } from '@/components/ui/card';
import { swrFetcher } from '@/lib/api/client';
import { uploadVideo } from '@/lib/api/videos';
import { suggestSlug } from '@/lib/utils/format';
import type { CategoryListResponse } from '@/lib/types';

export default function NewVideoPage() {
	const router = useRouter();
	const { data: cats, isLoading: catsLoading } = useSWR<CategoryListResponse>('/api/v1/admin/categories?limit=100&status=active', swrFetcher);

	const [title, setTitle] = useState('');
	const [slug, setSlug] = useState('');
	const [slugTouched, setSlugTouched] = useState(false);
	const [description, setDescription] = useState('');
	const [categoryIds, setCategoryIds] = useState<string[]>([]);
	const [visibility, setVisibility] = useState<'public' | 'private'>('private');
	const [thumbSecond, setThumbSecond] = useState('1');
	const [file, setFile] = useState<File | null>(null);

	const [pending, setPending] = useState(false);
	const [progress, setProgress] = useState(0);

	const canSubmit = !pending && title.trim() && slug.trim() && categoryIds.length > 0 && file !== null;

	function onTitleChange(v: string) {
		setTitle(v);
		if (!slugTouched) {
			setSlug(suggestSlug(v));
		}
	}

	async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		if (!canSubmit || !file) return;
		setPending(true);
		setProgress(0);
		try {
			const v = await uploadVideo({
				title: title.trim(),
				slug: slug.trim(),
				description: description.trim() || undefined,
				category_id: categoryIds[0],
				category_ids: categoryIds,
				visibility,
				thumbnail_second: thumbSecond === '' ? undefined : Number(thumbSecond),
				video: file,
				onProgress: (p) => setProgress(p),
			});
			toast.success('Upload complete');
			router.push(`/videos/${v.id}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Upload failed');
		} finally {
			setPending(false);
		}
	}

	return (
		<>
			<div className="mb-2">
				<Button asChild variant="ghost" size="sm">
					<Link href="/videos"><ArrowLeft className="h-4 w-4" /> Back to videos</Link>
				</Button>
			</div>
			<PageHeader title="Upload video" description="Multipart upload. Thumbnail is generated automatically via ffmpeg at the requested second." />

			<form onSubmit={handleSubmit} className="grid gap-6 lg:grid-cols-3">
				<div className="space-y-4 lg:col-span-2">
					<Card>
						<CardContent className="p-4">
							<UploadDropzone file={file} onFileChange={setFile} disabled={pending} />
						</CardContent>
					</Card>

					{pending && (
						<Card>
							<CardContent className="space-y-2 p-4">
								<div className="flex items-center justify-between text-sm">
									<span className="font-medium">Uploading…</span>
									<span className="tabular-nums">{progress}%</span>
								</div>
								<div className="h-2 w-full overflow-hidden rounded-full bg-muted">
									<div
										className="h-full bg-primary transition-all"
										style={{ width: `${progress}%` }}
									/>
								</div>
							</CardContent>
						</Card>
					)}
				</div>

				<div className="space-y-4">
					<Card>
						<CardContent className="space-y-4 p-4">
							<div className="space-y-1.5">
								<Label htmlFor="title">Title</Label>
								<Input id="title" required value={title} onChange={(e) => onTitleChange(e.target.value)} disabled={pending} />
							</div>
							<div className="space-y-1.5">
								<Label htmlFor="slug">Slug</Label>
								<Input
									id="slug"
									required
									value={slug}
									onChange={(e) => {
										setSlugTouched(true);
										setSlug(e.target.value);
									}}
									disabled={pending}
								/>
								<p className="text-xs text-muted-foreground">Normalized server-side (lower-case, kebab-case).</p>
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
									disabled={pending || catsLoading}
								/>
							</div>
							<div className="space-y-1.5">
								<Label>Visibility</Label>
								<Select value={visibility} onValueChange={(v) => setVisibility(v as 'public' | 'private')} disabled={pending}>
									<SelectTrigger><SelectValue /></SelectTrigger>
									<SelectContent>
										<SelectItem value="private">Private</SelectItem>
										<SelectItem value="public">Public</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div className="space-y-1.5">
								<Label htmlFor="thumb-sec">Thumbnail second</Label>
								<Input
									id="thumb-sec"
									type="number"
									min={0}
									max={86400}
									value={thumbSecond}
									onChange={(e) => setThumbSecond(e.target.value)}
									disabled={pending}
								/>
								<p className="text-xs text-muted-foreground">
									Frame to capture for the thumbnail. Default 1. If this value is
									longer than the video duration, the backend automatically uses
									the nearest valid second and returns it in the response.
								</p>
							</div>
						</CardContent>
					</Card>

					<Button type="submit" className="w-full" disabled={!canSubmit}>
						{pending ? (
							<><Loader2 className="h-4 w-4 animate-spin" /> Uploading…</>
						) : (
							'Upload'
						)}
					</Button>
				</div>
			</form>
		</>
	);
}
