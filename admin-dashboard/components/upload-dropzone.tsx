'use client';

import * as React from 'react';
import { UploadCloud, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { formatBytes } from '@/lib/utils/format';
import { cn } from '@/lib/utils/cn';

const ACCEPTED_TYPES = ['video/mp4', 'video/webm', 'video/quicktime'];
const ACCEPTED_LABEL = '.mp4, .webm, .mov';

interface UploadDropzoneProps {
	file: File | null;
	onFileChange: (file: File | null) => void;
	maxBytes?: number;
	disabled?: boolean;
}

export function UploadDropzone({ file, onFileChange, maxBytes, disabled }: UploadDropzoneProps) {
	const inputRef = React.useRef<HTMLInputElement | null>(null);
	const [dragOver, setDragOver] = React.useState(false);
	const [error, setError] = React.useState<string | null>(null);

	function validate(f: File): string | null {
		// Client-side MIME check is best-effort — the backend does the real
		// sniff. This catches the obvious cases before bytes go on the wire.
		if (!ACCEPTED_TYPES.includes(f.type) && f.type !== '') {
			return `Unsupported file type "${f.type}". Allowed: ${ACCEPTED_LABEL}`;
		}
		if (maxBytes && f.size > maxBytes) {
			return `File is too large (${formatBytes(f.size)}). Limit is ${formatBytes(maxBytes)}.`;
		}
		return null;
	}

	function pick(f: File | null) {
		if (!f) {
			onFileChange(null);
			setError(null);
			return;
		}
		const err = validate(f);
		if (err) {
			setError(err);
			onFileChange(null);
			return;
		}
		setError(null);
		onFileChange(f);
	}

	return (
		<div>
			<div
				role="button"
				tabIndex={0}
				aria-label="Choose video file"
				className={cn(
					'relative flex min-h-[140px] cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border-2 border-dashed p-6 text-center transition-colors',
					dragOver ? 'border-ring bg-accent' : 'border-input hover:bg-accent',
					disabled && 'pointer-events-none bg-zinc-100 text-zinc-500 dark:bg-zinc-800',
				)}
				onClick={() => inputRef.current?.click()}
				onKeyDown={(e) => {
					if (e.key === 'Enter' || e.key === ' ') {
						e.preventDefault();
						inputRef.current?.click();
					}
				}}
				onDragOver={(e) => {
					e.preventDefault();
					setDragOver(true);
				}}
				onDragLeave={() => setDragOver(false)}
				onDrop={(e) => {
					e.preventDefault();
					setDragOver(false);
					pick(e.dataTransfer.files?.[0] ?? null);
				}}
			>
				<input
					ref={inputRef}
					type="file"
					accept={ACCEPTED_TYPES.join(',')}
					className="hidden"
					onChange={(e) => pick(e.target.files?.[0] ?? null)}
				/>
				<UploadCloud className="h-8 w-8 text-muted-foreground" />
				<div className="text-sm font-medium">
					{file ? 'Replace file' : 'Drop a video here, or click to browse'}
				</div>
				<div className="text-xs text-muted-foreground">
					{ACCEPTED_LABEL}
					{maxBytes ? ` · up to ${formatBytes(maxBytes)}` : ''}
				</div>
			</div>

			{file && (
				<div className="mt-3 flex items-center justify-between gap-3 rounded-md border bg-card p-3 text-sm">
					<div className="min-w-0 flex-1">
						<div className="truncate font-medium">{file.name}</div>
						<div className="text-xs text-muted-foreground">
							{formatBytes(file.size)}
							{file.type ? ` · ${file.type}` : ''}
						</div>
					</div>
					<Button
						variant="ghost"
						size="icon"
						aria-label="Remove file"
						onClick={(e) => {
							e.stopPropagation();
							pick(null);
						}}
					>
						<X className="h-4 w-4" />
					</Button>
				</div>
			)}

			{error && <p className="mt-2 text-sm text-destructive">{error}</p>}
		</div>
	);
}
