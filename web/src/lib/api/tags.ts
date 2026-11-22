// Tag endpoints (plan 5.10, 5.14).
import { request } from './client.js';
import type { Tag, TagColor } from './types.js';

/** GET /api/tags */
export function listTags(): Promise<{ items: Tag[] }> {
	return request<{ items: Tag[] }>('GET', '/tags');
}

/** POST /api/tags */
export function createTag(name: string, color?: TagColor): Promise<Tag> {
	const body: { name: string; color?: TagColor } = { name };
	if (color !== undefined) body.color = color;
	return request<Tag>('POST', '/tags', { body });
}

/** PATCH /api/tags/{id} */
export function updateTag(id: number, patch: { name?: string; color?: TagColor }): Promise<Tag> {
	return request<Tag>('PATCH', `/tags/${id}`, { body: patch });
}

/** DELETE /api/tags/{id} */
export function deleteTag(id: number): Promise<void> {
	return request<void>('DELETE', `/tags/${id}`);
}
