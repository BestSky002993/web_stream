// Response shapes mirroring the Go backend.

export interface Admin {
	id: string;
	email: string;
	display_name: string;
	role: string;
	status: string;
}

export interface LoginResponse {
	token: string;
	expires_in: number;
	admin: {
		id: string;
		email: string;
		display_name: string;
		role: string;
	};
}

export interface Category {
	id: string;
	name: string;
	slug: string;
	description: string | null;
	status: 'active' | 'disabled';
	sort_order: number;
	created_at: string;
	updated_at: string;
}

export interface Pagination {
	page: number;
	limit: number;
	total: number;
	total_pages: number;
}

export interface CategoryListResponse {
	data: Category[];
	pagination: Pagination;
}

export type VideoStatus = 'draft' | 'processing' | 'ready' | 'blocked' | 'deleted';
export type VideoVisibility = 'public' | 'private';
export type HLSStatus = 'pending' | 'processing' | 'ready' | 'failed' | 'skipped';

export interface Video {
	id: string;
	title: string;
	slug: string;
	description: string | null;
	status: VideoStatus;
	visibility: VideoVisibility;
	category: { id: string; name: string };
	video: {
		filename: string;
		original_name: string;
		mime_type: string;
		size_bytes: number;
	};
	thumbnail: { filename: string; second: number };
	thumbnail_url: string;
	duration_seconds: number | null;
	views_count: number;
	uploaded_by_admin_id: string;
	hls: {
		status: HLSStatus;
		ready_at: string | null;
		error: string | null;
		variants: string[] | null;
	};
	approved_site_count: number;
	created_at: string;
	updated_at: string;
}

export interface SignedHLSURL {
	hls_url: string;
	expires_at: string;
	expires: number;
}

export interface VideoListResponse {
	data: Video[];
	pagination: Pagination;
}

export interface SignedStreamURL {
	stream_url: string;
	expires_at: string;
	expires: number;
}

export type SiteStatus = 'active' | 'disabled' | 'suspended';

export interface Site {
	id: string;
	name: string;
	domain: string;
	status: SiteStatus;
	daily_request_limit: number;
	api_key?: string; // only present on create / rotate
	created_at: string;
	updated_at: string;
}

export interface SiteListResponse {
	data: Site[];
	pagination: Pagination;
}

export type PublicationStatus = 'draft' | 'approved' | 'synced' | 'disabled' | 'failed' | 'blocked';

export interface SiteVideoPublication {
	id: string;
	site_id: string;
	video_id: string;
	status: PublicationStatus;
	sync_mode: SyncMode | null;
	remote_post_id: string | null;
	remote_slug: string | null;
	category_mapping: string | null;
	tags: string[];
	notes: string | null;
	last_synced_at: string | null;
	video?: {
		id: string;
		title: string;
		slug: string;
		description: string | null;
		category: { id: string; name: string };
		status: VideoStatus;
		visibility: VideoVisibility;
		duration_seconds: number | null;
		updated_at: string;
	};
	created_at: string;
	updated_at: string;
}

export interface PublicationListResponse {
	data: SiteVideoPublication[];
	pagination: Pagination;
}

export type AllowedDomainType = 'stream' | 'cors' | 'both';
export type AllowedDomainStatus = 'active' | 'disabled';

export interface AllowedDomain {
	id: string;
	domain: string;
	type: AllowedDomainType;
	status: AllowedDomainStatus;
	notes: string | null;
	created_at: string;
	updated_at: string;
}

export interface AllowedDomainListResponse {
	data: AllowedDomain[];
	pagination: Pagination;
}

export interface ApiError {
	status: number;
	message: string;
}

export type SyncMode = 'publish' | 'draft' | 'block';
export type CategoryMappingStatus = 'mapped' | 'unmapped' | 'disabled' | 'unconfirmed';

export interface RemoteCategory {
	id: string;
	site_id: string;
	remote_term_id: string;
	remote_slug: string;
	remote_name: string;
	remote_taxonomy: string;
	parent_remote_term_id: string | null;
	created_at: string;
	updated_at: string;
}

export interface RemoteCategoryListResponse {
	data: RemoteCategory[];
}

export interface CategoryMappingRow {
	backend_category: { id: string; name: string; slug: string };
	mapping_id: string | null;
	remote_category_id: string | null;
	remote_category: RemoteCategory | null;
	enabled: boolean;
	sync_mode: SyncMode;
	created_by: string | null;
	confirmed_by_wp: boolean;
	confirmed_by_stream: boolean;
	status: CategoryMappingStatus;
	created_at: string | null;
	updated_at: string | null;
}

export interface CategoryMappingListResponse {
	data: CategoryMappingRow[];
}

export interface SiteDistributionRow {
	site_id: string;
	site_name: string;
	site_domain: string;
	site_status: SiteStatus;
	publication_id: string | null;
	status: 'draft' | 'approved' | 'blocked';
	sync_mode: SyncMode;
	effective_sync_mode: SyncMode;
	mapping_status: CategoryMappingStatus;
	mapping_sync_mode: SyncMode;
	mapping_enabled: boolean;
	remote_category_id: string | null;
	remote_category_name: string | null;
	warnings: string[];
	updated_at: string | null;
	publication?: SiteVideoPublication;
}

export interface SiteDistributionResponse {
	video_id: string;
	data: SiteDistributionRow[];
}

export interface BulkDistributionResponse {
	approved: number;
	updated: number;
	skipped: number;
	errors: Array<{ video_id?: string; error: string }>;
	warnings?: Array<{ video_id: string; warning: string }>;
}
