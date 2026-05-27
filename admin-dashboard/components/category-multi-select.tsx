'use client';

import { Check, X } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils/cn';
import type { Category } from '@/lib/types';

interface CategoryMultiSelectProps {
	categories: Category[];
	value: string[];
	onChange: (ids: string[]) => void;
	disabled?: boolean;
}

export function CategoryMultiSelect({ categories, value, onChange, disabled }: CategoryMultiSelectProps) {
	const selected = new Set(value);
	function toggle(id: string) {
		if (disabled) return;
		if (selected.has(id)) {
			onChange(value.filter((v) => v !== id));
		} else {
			onChange([...value, id]);
		}
	}
	return (
		<div className="space-y-2">
			<div className="flex min-h-10 flex-wrap gap-1.5 rounded-md border border-[#374151] bg-[#111827] p-2">
				{value.length === 0 ? (
					<span className="px-1 text-sm text-[#9ca3af]">Select categories</span>
				) : value.map((id) => {
					const cat = categories.find((c) => c.id === id);
					return (
						<Badge key={id} variant="secondary" className="gap-1 bg-[#1f2937] text-[#f9fafb]">
							{cat?.name ?? id}
							<button type="button" onClick={() => toggle(id)} disabled={disabled} aria-label={`Remove ${cat?.name ?? id}`}>
								<X className="h-3 w-3" />
							</button>
						</Badge>
					);
				})}
			</div>
			<div className="max-h-48 overflow-y-auto rounded-md border border-[#374151] bg-[#111827] p-1">
				{categories.map((cat) => {
					const active = selected.has(cat.id);
					return (
						<button
							key={cat.id}
							type="button"
							onClick={() => toggle(cat.id)}
							disabled={disabled}
							className={cn(
								'flex w-full items-center justify-between border-b border-[#374151] px-2 py-2 text-left text-sm text-[#f9fafb] last:border-b-0 hover:bg-[#1f2937] focus:outline-none focus:ring-2 focus:ring-inset focus:ring-blue-400 disabled:cursor-not-allowed disabled:text-gray-300',
								active && 'bg-[#2563eb] font-medium hover:bg-blue-500',
							)}
						>
							<span>{cat.name}</span>
							{active && <Check className="h-4 w-4" />}
						</button>
					);
				})}
			</div>
			{value.length > 0 && (
				<Button type="button" variant="ghost" size="sm" onClick={() => onChange([])} disabled={disabled}>
					Clear categories
				</Button>
			)}
		</div>
	);
}
