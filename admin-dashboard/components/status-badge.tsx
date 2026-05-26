import { Badge } from '@/components/ui/badge';

interface StatusBadgeProps {
	value: string;
}

/**
 * Maps a backend status string to a coloured badge. Covers every status
 * we currently surface (video, category, site).
 */
export function StatusBadge({ value }: StatusBadgeProps) {
	const v = (value || '').toLowerCase();
	switch (v) {
		case 'ready':
		case 'active':
		case 'public':
			return <Badge variant="success">{value}</Badge>;
		case 'processing':
		case 'draft':
			return <Badge variant="warning">{value}</Badge>;
		case 'blocked':
		case 'disabled':
		case 'suspended':
		case 'deleted':
			return <Badge variant="destructive">{value}</Badge>;
		case 'private':
			return <Badge variant="muted">{value}</Badge>;
		default:
			return <Badge variant="outline">{value || '—'}</Badge>;
	}
}
