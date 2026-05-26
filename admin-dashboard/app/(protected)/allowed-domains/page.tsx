'use client';

import { useState } from 'react';
import useSWR, { mutate } from 'swr';
import { toast } from 'sonner';
import { MoreHorizontal, Pencil, Plus, PowerOff, ShieldCheck, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/page-header';
import { StatusBadge } from '@/components/status-badge';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Input, Textarea } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { swrFetcher } from '@/lib/api/client';
import { createAllowedDomain, deleteAllowedDomain, disableAllowedDomain, updateAllowedDomain } from '@/lib/api/allowed-domains';
import { formatDate } from '@/lib/utils/format';
import type { AllowedDomain, AllowedDomainListResponse, AllowedDomainStatus, AllowedDomainType } from '@/lib/types';

export default function AllowedDomainsPage() {
	const { data, isLoading } = useSWR<AllowedDomainListResponse>('/api/v1/admin/allowed-domains?limit=100', swrFetcher);
	const [editing, setEditing] = useState<AllowedDomain | 'new' | null>(null);
	const [disableTarget, setDisableTarget] = useState<AllowedDomain | null>(null);
	const [deleteTarget, setDeleteTarget] = useState<AllowedDomain | null>(null);
	const [pendingAction, setPendingAction] = useState(false);

	function refresh() {
		mutate((k) => typeof k === 'string' && k.startsWith('/api/v1/admin/allowed-domains'));
	}

	async function confirmDisable() {
		if (!disableTarget) return;
		setPendingAction(true);
		try {
			await disableAllowedDomain(disableTarget.id);
			toast.success('Domain disabled');
			setDisableTarget(null);
			refresh();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Disable failed');
		} finally {
			setPendingAction(false);
		}
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		setPendingAction(true);
		try {
			await deleteAllowedDomain(deleteTarget.id);
			toast.success('Domain deleted');
			setDeleteTarget(null);
			refresh();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Delete failed');
		} finally {
			setPendingAction(false);
		}
	}

	const items = data?.data ?? [];

	return (
		<>
			<PageHeader
				title="Allowed Domains"
				description="Hosts allowed for CORS and video hotlink checks."
				actions={<Button onClick={() => setEditing('new')}><Plus className="h-4 w-4" /> Add domain</Button>}
			/>

			{isLoading ? (
				<Skeleton className="h-48" />
			) : items.length === 0 ? (
				<EmptyState
					icon={<ShieldCheck className="h-8 w-8" />}
					title="No allowed domains"
					description="Add a WordPress or frontend host to allow browser API calls and video playback."
					action={<Button onClick={() => setEditing('new')}>Add domain</Button>}
				/>
			) : (
				<div className="rounded-lg border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Domain</TableHead>
								<TableHead>Type</TableHead>
								<TableHead>Status</TableHead>
								<TableHead>Notes</TableHead>
								<TableHead>Created</TableHead>
								<TableHead className="w-10" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{items.map((d) => (
								<TableRow key={d.id}>
									<TableCell><code className="text-xs">{d.domain}</code></TableCell>
									<TableCell className="capitalize">{d.type}</TableCell>
									<TableCell><StatusBadge value={d.status} /></TableCell>
									<TableCell className="max-w-sm truncate text-sm text-muted-foreground">{d.notes || '—'}</TableCell>
									<TableCell className="text-sm text-muted-foreground">{formatDate(d.created_at)}</TableCell>
									<TableCell>
										<DropdownMenu>
											<DropdownMenuTrigger asChild>
												<Button variant="ghost" size="icon" aria-label="Actions">
													<MoreHorizontal className="h-4 w-4" />
												</Button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end">
												<DropdownMenuItem onSelect={() => setEditing(d)}>
													<Pencil className="h-4 w-4" /> Edit
												</DropdownMenuItem>
												{d.status !== 'disabled' && (
													<DropdownMenuItem onSelect={() => setDisableTarget(d)}>
														<PowerOff className="h-4 w-4" /> Disable
													</DropdownMenuItem>
												)}
												<DropdownMenuSeparator />
												<DropdownMenuItem destructive onSelect={() => setDeleteTarget(d)}>
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

			<AllowedDomainDialog
				open={!!editing}
				value={editing === 'new' ? null : editing}
				onOpenChange={(o) => !o && setEditing(null)}
				onSaved={() => {
					setEditing(null);
					refresh();
				}}
			/>

			<ConfirmDialog
				open={!!disableTarget}
				onOpenChange={(o) => !o && setDisableTarget(null)}
				title={`Disable "${disableTarget?.domain}"?`}
				description="Requests from this host will stop passing CORS and stream hotlink checks."
				confirmLabel="Disable"
				destructive
				pending={pendingAction}
				onConfirm={confirmDisable}
			/>

			<ConfirmDialog
				open={!!deleteTarget}
				onOpenChange={(o) => !o && setDeleteTarget(null)}
				title={`Delete "${deleteTarget?.domain}"?`}
				description="This soft-deletes the allowlist entry and frees the domain for reuse."
				confirmLabel="Delete"
				destructive
				pending={pendingAction}
				onConfirm={confirmDelete}
			/>
		</>
	);
}

function AllowedDomainDialog({
	open,
	value,
	onOpenChange,
	onSaved,
}: {
	open: boolean;
	value: AllowedDomain | null;
	onOpenChange: (open: boolean) => void;
	onSaved: () => void;
}) {
	const isEdit = !!value;
	const [domain, setDomain] = useState('');
	const [type, setType] = useState<AllowedDomainType>('both');
	const [status, setStatus] = useState<AllowedDomainStatus>('active');
	const [notes, setNotes] = useState('');
	const [pending, setPending] = useState(false);

	const openKey = `${open}|${value?.id ?? 'new'}`;
	const [seededFor, setSeededFor] = useState('');
	if (open && openKey !== seededFor) {
		setDomain(value?.domain ?? '');
		setType(value?.type ?? 'both');
		setStatus(value?.status ?? 'active');
		setNotes(value?.notes ?? '');
		setSeededFor(openKey);
	}

	async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
		e.preventDefault();
		setPending(true);
		const payload = {
			domain: domain.trim(),
			type,
			status,
			notes: notes.trim() || undefined,
		};
		try {
			if (isEdit && value) {
				await updateAllowedDomain(value.id, payload);
				toast.success('Allowed domain updated');
			} else {
				await createAllowedDomain(payload);
				toast.success('Allowed domain added');
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
					<DialogTitle>{isEdit ? 'Edit allowed domain' : 'Add allowed domain'}</DialogTitle>
					<DialogDescription>
						Enter a host only, such as example.com, localhost, or testplugin.local.
					</DialogDescription>
				</DialogHeader>
				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="space-y-1.5">
						<Label htmlFor="allowed-domain">Domain</Label>
						<Input
							id="allowed-domain"
							required
							value={domain}
							onChange={(e) => setDomain(e.target.value)}
							placeholder="testplugin.local"
							disabled={pending}
						/>
					</div>
					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<Label>Type</Label>
							<Select value={type} onValueChange={(v) => setType(v as AllowedDomainType)} disabled={pending}>
								<SelectTrigger><SelectValue /></SelectTrigger>
								<SelectContent>
									<SelectItem value="both">Both</SelectItem>
									<SelectItem value="stream">Stream</SelectItem>
									<SelectItem value="cors">CORS</SelectItem>
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1.5">
							<Label>Status</Label>
							<Select value={status} onValueChange={(v) => setStatus(v as AllowedDomainStatus)} disabled={pending}>
								<SelectTrigger><SelectValue /></SelectTrigger>
								<SelectContent>
									<SelectItem value="active">Active</SelectItem>
									<SelectItem value="disabled">Disabled</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</div>
					<div className="space-y-1.5">
						<Label htmlFor="allowed-domain-notes">Notes</Label>
						<Textarea id="allowed-domain-notes" value={notes} onChange={(e) => setNotes(e.target.value)} disabled={pending} />
					</div>
					<DialogFooter>
						<Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>Cancel</Button>
						<Button type="submit" disabled={pending}>{pending ? 'Saving…' : isEdit ? 'Save changes' : 'Add domain'}</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
