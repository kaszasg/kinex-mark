// User directory for the share dialog (plan 5.11, 5.14).
import { request } from './client.js';
import type { DirectoryUser } from './types.js';

/** GET /api/users/directory?q= */
export function searchDirectory(
	q: string,
	signal?: AbortSignal
): Promise<{ items: DirectoryUser[] }> {
	return request<{ items: DirectoryUser[] }>('GET', '/users/directory', { query: { q }, signal });
}
