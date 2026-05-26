'use client';

import Link from 'next/link';
import { useState } from 'react';
import useSWR, { mutate } from 'swr';
import { toast } from 'sonner';
import { Globe, KeyRound, MoreHorizontal, Pencil, Plus, PowerOff, RefreshCw, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/page-header';
import { StatusBadge } from '@/components/status-badge';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { APIKeyModal } from '@/components/api-key-modal';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { swrFetcher } from '@/lib/api/client';
import { createSite, deleteSite, disableSite, rotateSiteKey, updateSite } from '@/lib/api/sites';
import { formatDate } from '@/lib/utils/format';
import type { Site, SiteListResponse } from '@/lib/types';

export default function SitesPage() {
	const { data, isLoading } = useSWR<SiteListResponse>('/api/v1/admin/sites?limit=100', swrFetcher);

	const [editing, setEditing] = useState<Site | 'new' | null>(null);
	const [keyResult, setKeyResult] = useState<{ apiKey: string; siteName: string } | null>(null);
	const [rotateTarget, setRotateTarget] = useState<Site | null>(null);
	const [rotatePending, setRotatePending] = useState(false);
	const [disableTarget, setDisableTarget] = useState<Site | null>(null);
	const [disablePending, setDisablePending] = useState(false);
	const [deleteTarget, setDeleteTarget] = useState<Site | null>(null);
	const [deletePending, setDeletePending] = useState(false);

	function refresh() {
		mutate((k) => typeof k === 'string' && k.startsWith('/api/v1/admin/sites'));
	}

	async function confirmRotate() {
		if (!rotateTarget) return;
		setRotatePending(true);
		try {
			const s = await rotateSiteKey(rotateTarget.id);
			toast.success('API key rotated. Old key has stopped working.');
			if (s.api_key) {
				setKeyResult({ apiKey: s.api_key, siteName: s.name });
			}
			setRotateTarget(null);
			refresh();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Rotate failed');
		} finally {
			setRotatePending(false);
		}
	}

	async function confirmDisable() {
		if (!disableTarget) return;
		setDisablePending(true);
		try {
			await disableSite(disableTarget.id);
			toast.success('Site disabled');
			setDisableTarget(null);
			refresh();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Disable failed');
		} finally {
			setDisablePending(false);
		}
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		setDeletePending(true);
		try {
			await deleteSite(deleteTarget.id);
			toast.success('Site deleted');
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
				title="Sites"
				description="External websites authorised to consume the API with a per-site key."
				actions={
					<Button onClick={() => setEditing('new')}>
						<Plus className="h-4 w-4" /> New site
					</Button>
				}
			/>

			{isLoading ? (
				<Skeleton className="h-48" />
			) : items.length === 0 ? (
				<EmptyState
					icon={<Globe className="h-8 w-8" />}
					title="No sites yet"
					description="Create one to issue an API key for a WordPress or other external frontend."
					action={<Button onClick={() => setEditing('new')}>Add site</Button>}
				/>
			) : (
				<div className="rounded-lg border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Domain</TableHead>
								<TableHead>Status</TableHead>
								<TableHead className="text-right">Daily limit</TableHead>
								<TableHead>Created</TableHead>
								<TableHead className="w-10" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{items.map((s) => (
								<TableRow key={s.id}>
									<TableCell className="font-medium">{s.name}</TableCell>
									<TableCell><code className="text-xs">{s.domain}</code></TableCell>
									<TableCell><StatusBadge value={s.status} /></TableCell>
									<TableCell className="text-right tabular-nums">{s.daily_request_limit.toLocaleString()}</TableCell>
									<TableCell className="text-sm text-muted-foreground">{formatDate(s.created_at)}</TableCell>
									<TableCell>
										<DropdownMenu>
											<DropdownMenuTrigger asChild>
												<Button variant="ghost" size="icon" aria-label="Actions">
													<MoreHorizontal className="h-4 w-4" />
												</Button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end">
												<DropdownMenuItem onSelect={() => setEditing(s)}>
													<Pencil className="h-4 w-4" /> Edit
												</DropdownMenuItem>
												<DropdownMenuItem asChild>
													<Link href={`/sites/${s.id}`}>Approved videos</Link>
												</DropdownMenuItem>
												<DropdownMenuItem onSelect={() => setRotateTarget(s)}>
													<RefreshCw className="h-4 w-4" /> Rotate API key
												</DropdownMenuItem>
												{s.status !== 'disabled' && (
													<DropdownMenuItem onSelect={() => setDisableTarget(s)}>
														<PowerOff className="h-4 w-4" /> Disable
													</DropdownMenuItem>
												)}
												<DropdownMenuSeparator />
												<DropdownMenuItem destructive onSelect={() => setDeleteTarget(s)}>
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

			<SiteFormDialog
				open={!!editing}
				value={editing === 'new' ? null : editing}
				onOpenChange={(o) => !o && setEditing(null)}
				onCreated={(s) => {
					setEditing(null);
					refresh();
					if (s.api_key) {
						setKeyResult({ apiKey: s.api_key, siteName: s.name });
					}
				}}
				onUpdated={() => {
					setEditing(null);
					refresh();
				}}
			/>

			<APIKeyModal
				apiKey={keyResult?.apiKey ?? null}
				siteName={keyResult?.siteName}
				onClose={() => setKeyResult(null)}
			/>

			<ConfirmDialog
				open={!!rotateTarget}
				onOpenChange={(o) => !o && setRotateTarget(null)}
				title={`Rotate API key for "${rotateTarget?.name}"?`}
				description="The existing key will stop working immediately. The new key is shown once after rotation."
				confirmLabel="Rotate key"
				destructive
				pending={rotatePending}
				onConfirm={confirmRotate}
			/>

			<ConfirmDialog
				open={!!disableTarget}
				onOpenChange={(o) => !o && setDisableTarget(null)}
				title={`Disable "${disableTarget?.name}"?`}
				description="Disabled sites are rejected with 403 even with a valid key. Re-enable from the edit dialog."
				confirmLabel="Disable"
				destructive
				pending={disablePending}
				onConfirm={confirmDisable}
			/>

			<ConfirmDialog
				open={!!deleteTarget}
				onOpenChange={(o) => !o && setDeleteTarget(null)}
				title={`Delete "${deleteTarget?.name}"?`}
				description="Soft-deleted on the backend. The key + domain become free for reuse."
				confirmLabel="Delete"
				destructive
				pending={deletePending}
				onConfirm={confirmDelete}
			/>
		</>
	);
}

interface SiteFormDialogProps {
	open: boolean;
	value: Site | null;
	onOpenChange: (open: boolean) => void;
	onCreated: (site: Site) => void;
	onUpdated: () => void;
}

function SiteFormDialog({ open, value, onOpenChange, onCreated, onUpdated }: SiteFormDialogProps) {
	const isEdit = !!value;
	const [name, setName] = useState('');
	const [domain, setDomain] = useState('');
	const [status, setStatus] = useState<'active' | 'disabled' | 'suspended'>('active');
	const [limit, setLimit] = useState<string>('10000');
	const [pending, setPending] = useState(false);

	const openKey = `${open}|${value?.id ?? 'new'}`;
	const [seededFor, setSeededFor] = useState('');
	if (open && openKey !== seededFor) {
		setName(value?.name ?? '');
		setDomain(value?.domain ?? '');
		setStatus((value?.status as 'active' | 'disabled' | 'suspended') ?? 'active');
		setLimit(String(value?.daily_request_limit ?? 10000));
		setSeededFor(openKey);
	}

	async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setPending(true);
		try {
			const payload = {
				name: name.trim(),
				domain: domain.trim(),
				status,
				daily_request_limit: Number(limit) || 0,
			};
			if (isEdit && value) {
				await updateSite(value.id, payload);
				toast.success('Site updated');
				onUpdated();
			} else {
				const s = await createSite(payload);
				toast.success('Site created');
				onCreated(s);
			}
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
					<DialogTitle>{isEdit ? 'Edit site' : 'New site'}</DialogTitle>
					<DialogDescription>
						{isEdit
							? 'Update this site\'s metadata. The API key is not editable here — use “Rotate API key”.'
							: 'A new API key is generated automatically. It is returned only once after creation.'}
					</DialogDescription>
				</DialogHeader>
				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="space-y-1.5">
						<Label htmlFor="site-name">Name</Label>
						<Input id="site-name" required value={name} onChange={(e) => setName(e.target.value)} disabled={pending} />
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="site-domain">Domain</Label>
						<Input
							id="site-domain"
							required
							value={domain}
							onChange={(e) => setDomain(e.target.value)}
							placeholder="example.com"
							disabled={pending}
						/>
						<p className="text-xs text-muted-foreground">
							Site requests must arrive from this domain (Origin / Referer) unless those headers are absent.
							Use <code>localhost</code> for dev WordPress installs.
						</p>
					</div>
					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<Label>Status</Label>
							<Select value={status} onValueChange={(v) => setStatus(v as never)} disabled={pending}>
								<SelectTrigger><SelectValue /></SelectTrigger>
								<SelectContent>
									<SelectItem value="active">Active</SelectItem>
									<SelectItem value="disabled">Disabled</SelectItem>
									<SelectItem value="suspended">Suspended</SelectItem>
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1.5">
							<Label htmlFor="site-limit">Daily request limit</Label>
							<Input
								id="site-limit"
								type="number"
								min={0}
								value={limit}
								onChange={(e) => setLimit(e.target.value)}
								disabled={pending}
							/>
						</div>
					</div>
					{!isEdit && (
						<div className="flex items-start gap-2 rounded-md border border-amber-700 bg-amber-950 p-3 text-sm text-amber-100">
							<KeyRound className="mt-0.5 h-4 w-4 shrink-0 text-amber-100" />
							<div>The API key is shown only once after this dialog closes. Be ready to copy it.</div>
						</div>
					)}
					<DialogFooter>
						<Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>Cancel</Button>
						<Button type="submit" disabled={pending}>{pending ? 'Saving…' : isEdit ? 'Save changes' : 'Create site'}</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
