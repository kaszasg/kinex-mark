// URL state of the users table (plan 8.7.7: ?q=&role=&status=&page=).
import type { AdminUserQuery } from '$lib/api/admin.js';
import type { Role } from '$lib/api/types.js';
import { ADMIN_PAGE_SIZE, pageOffset, parsePage } from './paging.js';

export const ROLES: readonly Role[] = ['admin', 'developer', 'customer'];
export type UserStatusFilter = 'active' | 'inactive' | 'all';
export const USER_STATUSES: readonly UserStatusFilter[] = ['active', 'inactive', 'all'];

export type UsersFilter = {
	q: string;
	role: Role | '';
	status: UserStatusFilter;
	page: number;
};

/** The default view: every role, active users, page 1. */
export const DEFAULT_USERS_FILTER: UsersFilter = { q: '', role: '', status: 'active', page: 1 };

const isRole = (value: string | null): value is Role => ROLES.includes(value as Role);
const isStatus = (value: string | null): value is UserStatusFilter =>
	USER_STATUSES.includes(value as UserStatusFilter);

/** Reads the filter from the query string; unknown values fall back to the defaults. */
export function parseUsersFilter(params: Pick<URLSearchParams, 'get'>): UsersFilter {
	const role = params.get('role');
	const status = params.get('status');
	return {
		q: (params.get('q') ?? '').trim().slice(0, 64),
		role: isRole(role) ? role : '',
		status: isStatus(status) ? status : 'active',
		page: parsePage(params.get('page'))
	};
}

/** The query string of a filter ('' for the default view); default values are left out. */
export function usersFilterSearch(filter: UsersFilter): string {
	const params = new URLSearchParams();
	if (filter.q) params.set('q', filter.q);
	if (filter.role) params.set('role', filter.role);
	if (filter.status !== 'active') params.set('status', filter.status);
	if (filter.page > 1) params.set('page', String(filter.page));
	const search = params.toString();
	return search ? `?${search}` : '';
}

/** True when any filter differs from the default view (the page number does not count). */
export function hasUserFilters(filter: UsersFilter): boolean {
	return filter.q !== '' || filter.role !== '' || filter.status !== 'active';
}

/** The GET /api/admin/users query of a filter. */
export function usersApiQuery(filter: UsersFilter): AdminUserQuery {
	return {
		q: filter.q || undefined,
		role: filter.role || undefined,
		status: filter.status,
		limit: ADMIN_PAGE_SIZE,
		offset: pageOffset(filter.page)
	};
}

/** A sign-in lock is shown while lockedUntil lies in the future. */
export function isLocked(lockedUntil: string | null, now: number = Date.now()): boolean {
	if (!lockedUntil) return false;
	const until = Date.parse(lockedUntil);
	return Number.isFinite(until) && until > now;
}
