'use client';

import { useState } from 'react';
import useSWR, { mutate } from 'swr';
import { toast } from 'sonner';
import { Folder, MoreHorizontal, Pencil, Plus, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/page-header';
import { StatusBadge } from '@/components/status-badge';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Input, Textarea } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { swrFetcher } from '@/lib/api/client';
import { createCategory, deleteCategory, updateCategory } from '@/lib/api/categories';
import { suggestSlug } from '@/lib/utils/format';
import type { Category, CategoryListResponse } from '@/lib/types';

export default function CategoriesPage() {
	const key = '/api/v1/admin/categories?limit=100';
	const { data, isLoading } = useSWR<CategoryListResponse>(key, swrFetcher);

	const [editing, setEditing] = useState<Category | 'new' | null>(null);
	const [deleteTarget, setDeleteTarget] = useState<Category | null>(null);
	const [deletePending, setDeletePending] = useState(false);

	function refresh() {
		mutate((k) => typeof k === 'string' && k.startsWith('/api/v1/admin/categories'));
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		setDeletePending(true);
		try {
			await deleteCategory(deleteTarget.id);
			toast.success('Category deleted');
			setDeleteTarget(null);
			refresh();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Delete failed');
		} finally {
			setDeletePending(false);
		}
	}

	const items = data?.data ?? [];

	return (
		<>
			<PageHeader
				title="Categories"
				description="Organize videos by category. Slugs power public URLs in future phases."
				actions={
					<Button onClick={() => setEditing('new')}>
						<Plus className="h-4 w-4" /> New category
					</Button>
				}
			/>

			{isLoading ? (
				<Skeleton className="h-48" />
			) : items.length === 0 ? (
				<EmptyState
					icon={<Folder className="h-8 w-8" />}
					title="No categories yet"
					description="Create one to group your videos."
					action={<Button onClick={() => setEditing('new')}>Create category</Button>}
				/>
			) : (
				<div className="rounded-lg border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Slug</TableHead>
								<TableHead>Status</TableHead>
								<TableHead className="text-right">Sort order</TableHead>
								<TableHead className="w-10" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{items.map((c) => (
								<TableRow key={c.id}>
									<TableCell className="font-medium">{c.name}</TableCell>
									<TableCell><code className="text-xs">{c.slug}</code></TableCell>
									<TableCell><StatusBadge value={c.status} /></TableCell>
									<TableCell className="text-right tabular-nums">{c.sort_order}</TableCell>
									<TableCell>
										<DropdownMenu>
											<DropdownMenuTrigger asChild>
												<Button variant="ghost" size="icon" aria-label="Actions">
													<MoreHorizontal className="h-4 w-4" />
												</Button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end">
												<DropdownMenuItem onSelect={() => setEditing(c)}>
													<Pencil className="h-4 w-4" /> Edit
												</DropdownMenuItem>
												<DropdownMenuItem destructive onSelect={() => setDeleteTarget(c)}>
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

			<CategoryFormDialog
				open={!!editing}
				value={editing === 'new' ? null : editing}
				onOpenChange={(o) => !o && setEditing(null)}
				onSaved={() => {
					setEditing(null);
					refresh();
				}}
			/>

			<ConfirmDialog
				open={!!deleteTarget}
				onOpenChange={(o) => !o && setDeleteTarget(null)}
				title={`Delete "${deleteTarget?.name}"?`}
				description="Soft-deleted on the backend. Videos referencing this category keep working (the FK is preserved)."
				confirmLabel="Delete"
				destructive
				pending={deletePending}
				onConfirm={confirmDelete}
			/>
		</>
	);
}

interface CategoryFormDialogProps {
	open: boolean;
	value: Category | null;
	onOpenChange: (open: boolean) => void;
	onSaved: () => void;
}

function CategoryFormDialog({ open, value, onOpenChange, onSaved }: CategoryFormDialogProps) {
	const isEdit = !!value;
	const [name, setName] = useState('');
	const [slug, setSlug] = useState('');
	const [slugTouched, setSlugTouched] = useState(false);
	const [description, setDescription] = useState('');
	const [status, setStatus] = useState<'active' | 'disabled'>('active');
	const [sortOrder, setSortOrder] = useState<string>('0');
	const [pending, setPending] = useState(false);

	// Reset form whenever the dialog re-opens for a different row.
	const editingId = value?.id ?? null;
	const openKey = `${open}|${editingId}`;
	const [seededFor, setSeededFor] = useState('');
	if (open && openKey !== seededFor) {
		setName(value?.name ?? '');
		setSlug(value?.slug ?? '');
		setSlugTouched(false);
		setDescription(value?.description ?? '');
		setStatus((value?.status as 'active' | 'disabled') ?? 'active');
		setSortOrder(String(value?.sort_order ?? 0));
		setSeededFor(openKey);
	}

	async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setPending(true);
		try {
			const payload = {
				name: name.trim(),
				slug: slug.trim(),
				description: description.trim() || undefined,
				status,
				sort_order: Number(sortOrder) || 0,
			};
			if (isEdit && value) {
				await updateCategory(value.id, payload);
				toast.success('Category updated');
			} else {
				await createCategory(payload);
				toast.success('Category created');
			}
			onSaved();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Save failed');
		} finally {
			setPending(false);
		}
	}

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle>{isEdit ? 'Edit category' : 'New category'}</DialogTitle>
					<DialogDescription>{isEdit ? 'Update this category.' : 'Add a new category for organizing videos.'}</DialogDescription>
				</DialogHeader>
				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="space-y-1.5">
						<Label htmlFor="cat-name">Name</Label>
						<Input
							id="cat-name"
							required
							value={name}
							onChange={(e) => {
								setName(e.target.value);
								if (!slugTouched) setSlug(suggestSlug(e.target.value));
							}}
							disabled={pending}
						/>
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="cat-slug">Slug</Label>
						<Input
							id="cat-slug"
							required
							value={slug}
							onChange={(e) => {
								setSlugTouched(true);
								setSlug(e.target.value);
							}}
							disabled={pending}
						/>
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="cat-desc">Description</Label>
						<Textarea id="cat-desc" value={description} onChange={(e) => setDescription(e.target.value)} disabled={pending} />
					</div>
					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<Label>Status</Label>
							<Select value={status} onValueChange={(v) => setStatus(v as 'active' | 'disabled')} disabled={pending}>
								<SelectTrigger><SelectValue /></SelectTrigger>
								<SelectContent>
									<SelectItem value="active">Active</SelectItem>
									<SelectItem value="disabled">Disabled</SelectItem>
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1.5">
							<Label htmlFor="cat-sort">Sort order</Label>
							<Input id="cat-sort" type="number" value={sortOrder} onChange={(e) => setSortOrder(e.target.value)} disabled={pending} />
						</div>
					</div>
					<DialogFooter>
						<Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>Cancel</Button>
						<Button type="submit" disabled={pending}>{pending ? 'Saving…' : isEdit ? 'Save changes' : 'Create'}</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
