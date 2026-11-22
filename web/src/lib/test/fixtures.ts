// Typed test data factories (plan 9.3.1, 9.7).
import type { ErrorCode, FieldCode } from '$lib/api/errors.js';
import type { LinkItem, Me, Preferences, PublicConfig, Role, Tag } from '$lib/api/types.js';

export const T0 = '2026-01-05T10:00:00.000Z';

let nextId = 1000;
function id(): number {
	nextId += 1;
	return nextId;
}

export function makePreferences(partial: Partial<Preferences> = {}): Preferences {
	return {
		locale: 'hu',
		theme: 'dark',
		viewMode: 'list',
		showIcon: true,
		showUrl: true,
		showTitle: true,
		showDescription: true,
		showArchived: false,
		sort: 'usage',
		tagMatch: 'any',
		archiveAfterDays: null,
		effectiveArchiveAfterDays: 90,
		systemArchiveAfterDays: 90,
		...partial
	};
}

const ROLE_PERMISSIONS: Record<Role, Me['permissions']> = {
	admin: { canShare: true, canListUsers: true, canAdmin: true },
	developer: { canShare: true, canListUsers: true, canAdmin: false },
	customer: { canShare: false, canListUsers: false, canAdmin: false }
};

/** A signed-in user; `preferences` may be partial and is merged over the defaults. */
export function makeMe(
	role: Role = 'developer',
	partial: Omit<Partial<Me>, 'preferences'> & { preferences?: Partial<Preferences> } = {}
): Me {
	const { preferences, ...rest } = partial;
	return {
		id: 1,
		username: 'kanna',
		displayName: 'Kovács Anna',
		role,
		mustChangePassword: false,
		totpEnabled: false,
		recoveryCodesRemaining: 0,
		privacyConsentRequired: false,
		privacyNoticeVersion: 1,
		permissions: ROLE_PERMISSIONS[role],
		sessionId: 1,
		createdAt: T0,
		lastLoginAt: T0,
		...rest,
		preferences: makePreferences(preferences)
	};
}

export function makeTag(partial: Partial<Tag> = {}): Tag {
	return {
		id: id(),
		name: 'infra',
		color: 'blue',
		linkCount: 3,
		activeLinkCount: 2,
		createdAt: T0,
		updatedAt: T0,
		...partial
	};
}

export function makeLink(partial: Partial<LinkItem> = {}): LinkItem {
	return {
		id: id(),
		url: 'https://grafana.example.net/d/abc',
		host: 'grafana.example.net',
		title: 'Grafana',
		description: 'Éles műszerfal',
		iconUrl: null,
		fetchStatus: 'ok',
		tagIds: [],
		pinned: false,
		useCount: 0,
		lastUsedAt: null,
		archivedAt: null,
		archiveReason: null,
		inactiveSince: T0,
		archiveDueAt: null,
		isOwner: true,
		owner: { id: 1, username: 'kanna', displayName: 'Kovács Anna' },
		shareCount: 0,
		sharedVia: null,
		sharedAt: null,
		createdAt: T0,
		updatedAt: T0,
		...partial
	};
}

export function makePublicConfig(partial: Partial<PublicConfig> = {}): PublicConfig {
	return {
		appName: 'Kinex Mark',
		version: '0.1.0',
		locales: ['hu', 'en', 'de'],
		defaultLocale: 'hu',
		tagColors: ['blue', 'ochre', 'green', 'violet', 'rose', 'neutral', 'contrast'],
		passwordPolicy: { minLength: 12, maxLength: 128, minDistinct: 6 },
		privacy: {
			controllerName: 'Kinexium',
			controllerAddress: '',
			controllerEmail: '',
			dpoContact: '',
			supervisoryAuthority: '',
			noticeVersion: 1,
			updatedAt: null,
			auditRetentionDays: 180,
			sessionIdleMinutes: 720,
			sessionAbsoluteHours: 168
		},
		...partial
	};
}

/** An error body as the server writes it (plan 5.2). */
export function apiErrorBody(
	code: ErrorCode,
	fields?: Record<string, FieldCode>,
	details?: Record<string, unknown>
): {
	error: string;
	code: ErrorCode;
	fields?: Record<string, FieldCode>;
	details?: Record<string, unknown>;
} {
	const body: {
		error: string;
		code: ErrorCode;
		fields?: Record<string, FieldCode>;
		details?: Record<string, unknown>;
	} = { error: code.replace(/_/g, ' '), code };
	if (fields) body.fields = fields;
	if (details) body.details = details;
	return body;
}
