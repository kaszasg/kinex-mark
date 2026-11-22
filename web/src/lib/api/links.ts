// Link endpoints (plan 5.9, 5.14).
import { download, request, type Query } from './client.js';
import type {
	ImportResult,
	LinkDetail,
	LinkItem,
	LinkPage,
	LinkPatch,
	LinkQuery,
	LinkState,
	NewLink,
	QuickCaptureResult,
	Share,
	ShareTarget
} from './types.js';

/** Maps a LinkQuery to the query parameters of plan 5.9.1. */
export function linkQueryParams(query: LinkQuery): Query {
	return {
		q: query.q,
		tags: query.tags && query.tags.length > 0 ? query.tags.join(',') : undefined,
		tagMatch: query.tagMatch,
		untagged: query.untagged === true ? true : undefined,
		scope: query.scope,
		archived: query.archived,
		sort: query.sort,
		limit: query.limit,
		offset: query.offset
	};
}

/** GET /api/links */
export function listLinks(query: LinkQuery, signal?: AbortSignal): Promise<LinkPage> {
	return request<LinkPage>('GET', '/links', { query: linkQueryParams(query), signal });
}

/** POST /api/links */
export function createLink(input: NewLink): Promise<LinkDetail> {
	return request<LinkDetail>('POST', '/links', { body: input });
}

/** POST /api/links/quick */
export function quickCapture(text: string, title?: string): Promise<QuickCaptureResult> {
	const body: { text: string; title?: string } = { text };
	if (title !== undefined && title !== '') body.title = title;
	return request<QuickCaptureResult>('POST', '/links/quick', { body });
}

/** GET /api/links/{id} */
export function getLink(id: number, signal?: AbortSignal): Promise<LinkDetail> {
	return request<LinkDetail>('GET', `/links/${id}`, { signal });
}

/** PATCH /api/links/{id} */
export function updateLink(id: number, patch: LinkPatch): Promise<LinkDetail> {
	return request<LinkDetail>('PATCH', `/links/${id}`, { body: patch });
}

/** DELETE /api/links/{id} */
export function deleteLink(id: number): Promise<void> {
	return request<void>('DELETE', `/links/${id}`);
}

/** PUT /api/links/{id}/tags */
export function setLinkTags(id: number, tagIds: number[]): Promise<LinkItem> {
	return request<LinkItem>('PUT', `/links/${id}/tags`, { body: { tagIds } });
}

/** POST /api/links/{id}/pin */
export function pin(id: number): Promise<LinkItem> {
	return request<LinkItem>('POST', `/links/${id}/pin`);
}

/** DELETE /api/links/{id}/pin */
export function unpin(id: number): Promise<LinkItem> {
	return request<LinkItem>('DELETE', `/links/${id}/pin`);
}

/** POST /api/links/{id}/archive */
export function archive(id: number): Promise<LinkItem> {
	return request<LinkItem>('POST', `/links/${id}/archive`);
}

/** DELETE /api/links/{id}/archive (restore) */
export function restore(id: number): Promise<LinkItem> {
	return request<LinkItem>('DELETE', `/links/${id}/archive`);
}

/** POST /api/links/{id}/visit, sent with keepalive so it survives the navigation it accompanies. */
export function recordVisit(id: number): Promise<LinkState> {
	return request<LinkState>('POST', `/links/${id}/visit`, { keepalive: true });
}

/** POST /api/links/{id}/refresh */
export function refreshMetadata(id: number): Promise<LinkDetail> {
	return request<LinkDetail>('POST', `/links/${id}/refresh`);
}

/** GET /api/links/{id}/shares */
export function listShares(id: number): Promise<{ items: Share[] }> {
	return request<{ items: Share[] }>('GET', `/links/${id}/shares`);
}

/** POST /api/links/{id}/shares */
export function createShare(id: number, target: ShareTarget): Promise<Share> {
	return request<Share>('POST', `/links/${id}/shares`, { body: target });
}

/** DELETE /api/links/{id}/shares/{shareId} */
export function deleteShare(id: number, shareId: number): Promise<void> {
	return request<void>('DELETE', `/links/${id}/shares/${shareId}`);
}

/** GET /api/links/export?format=json|html (file download) */
export function exportLinks(format: 'json' | 'html'): Promise<{ blob: Blob; filename: string }> {
	return download('/links/export', { format });
}

/** The import content type, chosen from the file extension (plan 8.7.3). */
export function importContentType(fileName: string): 'application/json' | 'text/html' {
	return /\.json$/i.test(fileName) ? 'application/json' : 'text/html';
}

/** POST /api/links/import with the raw file as the body. */
export function importLinks(file: File): Promise<ImportResult> {
	return request<ImportResult>('POST', '/links/import', {
		raw: file,
		contentType: importContentType(file.name)
	});
}
