'use client';

import * as React from 'react';
import { toast } from 'sonner';
import { Copy, Check, AlertTriangle } from 'lucide-react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';

interface APIKeyModalProps {
	apiKey: string | null;
	siteName?: string;
	onClose: () => void;
}

/**
 * Shown exactly once after creating or rotating a site's API key. The
 * backend never returns the plain key again, so this is the only chance
 * to copy it.
 */
export function APIKeyModal({ apiKey, siteName, onClose }: APIKeyModalProps) {
	const [copied, setCopied] = React.useState(false);

	async function copy() {
		if (!apiKey) return;
		try {
			await navigator.clipboard.writeText(apiKey);
			setCopied(true);
			toast.success('API key copied to clipboard');
			setTimeout(() => setCopied(false), 2000);
		} catch {
			toast.error('Could not copy — copy it manually');
		}
	}

	return (
		<Dialog open={!!apiKey} onOpenChange={(o) => !o && onClose()}>
			<DialogContent className="sm:max-w-lg" showClose={false}>
				<DialogHeader>
					<DialogTitle>API key for {siteName || 'this site'}</DialogTitle>
					<DialogDescription>
						This key is shown <strong>only once</strong>. Copy it now — the backend does not store the plain value and you cannot retrieve it later. If you lose it, rotate the key to get a new one.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-3">
					<div className="flex items-start gap-2 rounded-md border border-amber-700 bg-amber-950 p-3 text-sm text-amber-100">
						<AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-100" />
						<div>Treat this key like a password. Anyone with it can call the public site endpoints from your configured domain.</div>
					</div>

					<div className="flex items-center gap-2 rounded-md border bg-muted p-3 font-mono text-xs">
						<div className="min-w-0 flex-1 break-all">{apiKey}</div>
						<Button size="icon" variant="ghost" onClick={copy} aria-label="Copy API key">
							{copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
						</Button>
					</div>
				</div>

				<DialogFooter>
					<Button onClick={onClose}>I have copied the key</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
