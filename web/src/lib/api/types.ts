// JSON shapes of the API contract, taken verbatim from plan 5.5 and 5.8.1.

export type Role = 'admin' | 'developer' | 'customer';
export type Locale = 'hu' | 'en' | 'de';
export type TagColor = 'blue' | 'ochre' | 'green' | 'violet' | 'rose' | 'neutral' | 'contrast';
export type FetchStatus = 'pending' | 'ok' | 'failed' | 'blocked';
export type FetchError =
	| 'timeout'
	| 'dns'
	| 'connection'
	| 'tls'
	| 'http_status'
	| 'too_large'
	| 'not_html'
	| 'too_many_redirects'
	| 'blocked_address'
	| 'blocked_port';
export type ShareTargetType = 'all' | 'role' | 'user';

export type UserRef = { id: number; username: string; displayName: string };
export type DirectoryUser = UserRef & { role: Role };

export type Preferences = {
	locale: Locale;
	theme: 'dark' | 'light' | 'system';
	viewMode: 'list' | 'tiles';
	showIcon: boolean;
	showUrl: boolean;
	showTitle: boolean;
	showDescription: boolean;
	showArchived: boolean;
	sort: 'usage' | 'created' | 'title';
	tagMatch: 'any' | 'all';
	archiveAfterDays: number | null; // null: the system default
	effectiveArchiveAfterDays: number; // read-only; 0 = never
	systemArchiveAfterDays: number; // read-only; the current defaultArchiveDays system setting
};

export type Me = {
	id: number;
	username: string;
	displayName: string;
	role: Role;
	mustChangePassword: boolean;
	totpEnabled: boolean;
	recoveryCodesRemaining: number;
	privacyConsentRequired: boolean;
	privacyNoticeVersion: number;
	permissions: { canShare: boolean; canListUsers: boolean; canAdmin: boolean };
	preferences: Preferences;
	sessionId: number;
	createdAt: string;
	lastLoginAt: string | null;
};

export type LinkItem = {
	id: number;
	url: string;
	host: string;
	title: string; // fetched or manually entered title, may be ''
	description: string; // short description, may be ''
	iconUrl: string | null; // "/api/links/12/icon?v=3fa2c1d0" when an icon exists
	fetchStatus: FetchStatus;
	tagIds: number[]; // the viewer's own tags only
	pinned: boolean;
	useCount: number;
	lastUsedAt: string | null;
	archivedAt: string | null;
	archiveReason: 'auto' | 'manual' | null;
	inactiveSince: string; // max(baselineAt, lastUsedAt): inactivity counts from here
	archiveDueAt: string | null; // inactiveSince + effective period; null when never, pinned or archived
	isOwner: boolean;
	owner: UserRef;
	shareCount: number; // non-zero only for the owner
	sharedVia: ShareTargetType | null; // recipients only; priority user > role > all
	sharedAt: string | null;
	createdAt: string;
	updatedAt: string;
};

export type LinkDetail = LinkItem & {
	titleSource: 'none' | 'fetched' | 'manual'; // section 4.4
	tags: Tag[]; // tagIds resolved
	fetchError: FetchError | null;
	fetchedAt: string | null;
	fetchAttempts: number;
	shares: Share[]; // owner with share permission only, otherwise []
};

export type LinkState = Pick<
	LinkItem,
	| 'pinned'
	| 'useCount'
	| 'lastUsedAt'
	| 'archivedAt'
	| 'archiveReason'
	| 'inactiveSince'
	| 'archiveDueAt'
>;

export type Tag = {
	id: number;
	name: string;
	color: TagColor;
	linkCount: number; // visible links carrying this tag
	activeLinkCount: number; // of which not archived
	createdAt: string;
	updatedAt: string;
};

export type Share = {
	id: number;
	targetType: ShareTargetType;
	targetRole: Role | null;
	targetUser: UserRef | null;
	createdAt: string;
};

export type Session = {
	id: number;
	createdAt: string;
	lastSeenAt: string;
	expiresAt: string;
	ip: string;
	userAgent: string;
	current: boolean;
};

export type AdminUser = {
	id: number;
	username: string;
	displayName: string;
	role: Role;
	active: boolean;
	mustChangePassword: boolean;
	totpEnabled: boolean;
	lockedUntil: string | null;
	lastLoginAt: string | null;
	createdAt: string;
	updatedAt: string;
	linkCount: number;
	sharedLinkCount: number; // own links with at least one share (shown in the delete dialog)
	sessionCount: number;
};

export type SystemSettings = {
	defaultArchiveDays: number; // 0-3650, default 90; 0 = never
	auditRetentionDays: number; // 30-3650, default 180
	sessionIdleMinutes: number; // 15-10080, default 720
	sessionAbsoluteHours: number; // 1-2160, default 168
	allowPrivateFetch: boolean; // default false
	privacyControllerName: string; // 0-200 runes
	privacyControllerAddress: string; // 0-300 runes
	privacyControllerEmail: string; // 0-254, empty or e-mail shaped
	privacyDpoContact: string; // 0-300 runes
	privacySupervisoryAuthority: string; // 0-300 runes; empty: the notice names the authority generically
	privacyNoticeVersion: number; // read-only, bumped by the bump endpoint
	privacyNoticeUpdatedAt: string | null; // read-only
};

export type AuditTargetType = 'user' | 'link' | 'session' | 'settings' | 'tag';
export type AuditEntry = {
	id: number;
	at: string;
	event: string;
	success: boolean;
	actor: { id: number | null; username: string };
	target: { type: AuditTargetType; id: number | null; name: string } | null;
	ip: string;
	details: Record<string, unknown>;
};

export type Page<T> = { items: T[]; total: number; limit: number; offset: number };

export type PublicConfig = {
	appName: 'Kinex Mark';
	version: string;
	locales: Locale[]; // ["hu","en","de"]
	defaultLocale: 'hu';
	tagColors: TagColor[]; // in the order of section 4.6
	passwordPolicy: { minLength: 12; maxLength: 128; minDistinct: 6 };
	privacy: {
		controllerName: string;
		controllerAddress: string;
		controllerEmail: string;
		dpoContact: string;
		supervisoryAuthority: string;
		noticeVersion: number;
		updatedAt: string | null;
		auditRetentionDays: number;
		sessionIdleMinutes: number;
		sessionAbsoluteHours: number;
	};
};

export type LoginResponse =
	{ status: 'ok'; user: Me } | { status: 'totp_required'; challenge: string; expiresAt: string };
export type TwoFactorSetup = { secret: string; otpauthUri: string; expiresAt: string };
export type RecoveryCodes = { recoveryCodes: string[] };
export type PrivacyStatus = {
	currentVersion: number;
	consentRequired: boolean; // acknowledgement of the current notice is missing
	consents: { kind: 'privacy_notice'; version: number; acceptedAt: string }[];
	ownSharedLinkCount: number; // own links with at least one share (account deletion dialog)
};
export type LinkPage = Page<LinkItem> & {
	counts: { pinned: number; archived: number }; // within the filtered set (section 4.8)
};
export type LinkQuery = {
	q?: string;
	tags?: number[];
	tagMatch?: 'any' | 'all';
	untagged?: boolean;
	scope?: 'all' | 'own' | 'shared';
	archived?: 'exclude' | 'include' | 'only';
	sort?: 'usage' | 'created' | 'title';
	limit?: number;
	offset?: number;
};
export type NewLink = {
	url: string;
	title?: string;
	description?: string;
	tagIds?: number[];
	pinned?: boolean;
};
export type LinkPatch = { url?: string; title?: string; description?: string };
export type QuickCaptureResult = { link: LinkDetail; created: boolean };
export type ShareTarget = { targetType: ShareTargetType; targetRole?: Role; targetUserId?: number };
export type ImportResult = { total: number; created: number; duplicates: number; invalid: number };
export type NewAdminUser = {
	username: string;
	displayName?: string;
	role: Role;
	password?: string;
	locale?: Locale;
};
export type AdminUserPatch = {
	displayName?: string;
	role?: Role;
	active?: boolean;
	currentPassword?: string; // the admin's own password; required when role or active is present
};
export type AdminStepUp = { currentPassword: string }; // reset-password, reset-2fa, delete
export type AdminUserCreated = { user: AdminUser; temporaryPassword: string | null };
export type AdminSettingsResponse = {
	settings: SystemSettings;
	defaults: SystemSettings;
	lockedByEnv: string[];
};
export type AuditQuery = {
	event?: string;
	actorId?: number;
	success?: boolean;
	from?: string;
	to?: string;
	limit?: number;
	offset?: number;
};

export type UserExport = {
	format: 'kinex-mark-export/1';
	exportedAt: string;
	user: {
		id: number;
		username: string;
		displayName: string;
		role: Role;
		active: boolean;
		totpEnabled: boolean;
		createdAt: string;
		lastLoginAt: string | null;
		passwordChangedAt: string;
	};
	preferences: Preferences;
	consents: { kind: string; version: number; acceptedAt: string }[];
	sessions: Omit<Session, 'current'>[];
	tags: { id: number; name: string; color: TagColor; createdAt: string }[];
	links: {
		// own links
		id: number;
		url: string;
		title: string;
		description: string;
		createdAt: string;
		updatedAt: string;
		tags: string[];
		pinned: boolean;
		useCount: number;
		lastUsedAt: string | null;
		archivedAt: string | null;
		shares: {
			targetType: ShareTargetType;
			targetRole: Role | null;
			targetUsername: string | null;
			createdAt: string;
		}[];
	}[];
	sharedWithMe: {
		// personal state on links owned by others
		linkId: number;
		url: string | null; // null: the link is no longer visible to the user (share revoked, owner demoted)
		tags: string[];
		pinned: boolean;
		useCount: number;
		lastUsedAt: string | null;
		archivedAt: string | null;
	}[];
	auditEvents: {
		// the user is the actor, or the target user
		at: string;
		event: string;
		success: boolean;
		ip: string;
		role: 'actor' | 'target';
	}[];
	security: { lockedUntil: string | null }; // current sign-in lock of the account
};
