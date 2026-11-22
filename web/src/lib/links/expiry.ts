// Time left until the automatic archiving of a link (plan 8.7.3 ExpiryMeter).
import type { LinkItem } from '$lib/api/types.js';

export const EXPIRY_SEGMENTS = 5;
/** At or below this many days the meter turns into a warning icon with a day count. */
export const EXPIRY_SOON_DAYS = 7;
const DAY_MS = 86_400_000;

export type ExpiryInfo = {
	/** Filled segments, 0..5; at least 1 while the due date is ahead. */
	segments: number;
	/** Whole days left, rounded up; 0 when due today or overdue. */
	daysLeft: number;
	soon: boolean;
};

export function expiryInfo(input: {
	archiveDueAt: string | null;
	inactiveSince: string;
	now: Date;
}): ExpiryInfo | null {
	if (!input.archiveDueAt) return null;
	const due = Date.parse(input.archiveDueAt);
	if (Number.isNaN(due)) return null;
	const start = Date.parse(input.inactiveSince);
	const remaining = due - input.now.getTime();
	const total = Number.isNaN(start) ? 0 : due - start;
	const daysLeft = remaining > 0 ? Math.ceil(remaining / DAY_MS) : 0;
	let segments = 0;
	if (remaining > 0) {
		segments =
			total > 0
				? Math.min(EXPIRY_SEGMENTS, Math.max(1, Math.ceil((EXPIRY_SEGMENTS * remaining) / total)))
				: EXPIRY_SEGMENTS;
	}
	return { segments, daysLeft, soon: daysLeft <= EXPIRY_SOON_DAYS };
}

/** The meter is hidden on pinned and archived links and when archiving is off (0 days). */
export function showsExpiry(
	link: Pick<LinkItem, 'pinned' | 'archivedAt' | 'archiveDueAt'>,
	effectiveArchiveAfterDays: number
): boolean {
	return (
		!link.pinned &&
		link.archivedAt === null &&
		effectiveArchiveAfterDays > 0 &&
		link.archiveDueAt !== null
	);
}
