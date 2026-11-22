// Page arithmetic shared by the admin tables (plan 5.4: offset = (page - 1) x limit).

/** Rows per page in the users and audit tables (plan 8.7.7, 8.7.9). */
export const ADMIN_PAGE_SIZE = 50;

/** Parses a 1-based page number from the URL; anything else is page 1. */
export function parsePage(raw: string | null): number {
	if (!raw || !/^\d{1,6}$/.test(raw)) return 1;
	const page = Number(raw);
	return page >= 1 ? page : 1;
}

/** The API offset of a 1-based page. */
export function pageOffset(page: number, size = ADMIN_PAGE_SIZE): number {
	return (Math.max(1, page) - 1) * size;
}

/** Number of pages for a total; at least 1, so an empty table still has page 1. */
export function pageCount(total: number, size = ADMIN_PAGE_SIZE): number {
	return Math.max(1, Math.ceil(total / size));
}

/** 1-based first and last row shown on a page ({from: 0, to: 0} for an empty result). */
export function pageRange(
	page: number,
	total: number,
	size = ADMIN_PAGE_SIZE
): { from: number; to: number } {
	if (total <= 0) return { from: 0, to: 0 };
	const from = Math.min(pageOffset(page, size) + 1, total);
	return { from, to: Math.min(from + size - 1, total) };
}
