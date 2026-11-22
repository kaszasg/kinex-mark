// Translation of audit events and their details (plan 8.5.8). Unknown codes stay raw, in mono.
import type { AuditEntry } from '$lib/api/types.js';
import { fmt, hasMessage, t } from '$lib/i18n/index.js';

/** A translated text, or the raw code when no translation exists (`raw` asks for mono). */
export type Label = { text: string; raw: boolean };

function lookup(key: string, fallback: string): Label {
	return hasMessage(key) ? { text: t(key), raw: false } : { text: fallback, raw: true };
}

/** audit.events.<event>, or the event code itself. */
export function eventLabel(event: string): Label {
	return lookup(`audit.events.${event}`, event);
}

/** audit.targets.<type>, or the type code itself. */
export function targetLabel(type: string): Label {
	return lookup(`audit.targets.${type}`, type);
}

export type DetailRow = { key: string; label: Label; value: Label };

const IDENTIFIER_KEYS = new Set(['sessionId', 'targetUserId']);
const COUNT_KEYS = new Set(['remaining', 'count', 'version']);

function raw(value: unknown): Label {
	if (value === null || value === undefined) return { text: fmt.missing, raw: false };
	return { text: typeof value === 'string' ? value : JSON.stringify(value), raw: true };
}

function bool(value: unknown): Label {
	if (typeof value !== 'boolean') return raw(value);
	return { text: t(value ? 'audit.details.yes' : 'audit.details.no'), raw: false };
}

function role(value: unknown): Label {
	if (value === null) return { text: fmt.missing, raw: false };
	if (typeof value !== 'string') return raw(value);
	return lookup(`roles.${value}`, value);
}

function enumerated(key: string, value: unknown): Label {
	if (typeof value !== 'string') return raw(value);
	return lookup(`audit.details.values.${key}.${value}`, value);
}

/** A {from, to} pair of the user.updated event, each side formatted by `side`. */
function change(value: unknown, side: (v: unknown) => Label): Label | null {
	if (typeof value !== 'object' || value === null || Array.isArray(value)) return null;
	const pair = value as Record<string, unknown>;
	if (!('from' in pair) || !('to' in pair)) return null;
	const from = side(pair.from);
	const to = side(pair.to);
	return {
		text: t('audit.details.change', { from: from.text, to: to.text }),
		raw: from.raw && to.raw
	};
}

/** Field names of the changed list: system settings or user dialog labels, else raw. */
function changedFields(event: string, value: unknown): Label {
	if (!Array.isArray(value)) return raw(value);
	const namespace = event.startsWith('settings.')
		? 'admin.settings.fields'
		: 'admin.userDialog.fields';
	const labels = value.map((field) => lookup(`${namespace}.${String(field)}`, String(field)));
	return {
		text: labels.map((label) => label.text).join(', '),
		raw: labels.length > 0 && labels.every((label) => label.raw)
	};
}

function number(value: unknown, grouped: boolean): Label {
	if (typeof value !== 'number') return raw(value);
	return { text: grouped ? fmt.number(value) : String(value), raw: false };
}

function detailValue(event: string, key: string, value: unknown): Label {
	switch (key) {
		case 'reason':
		case 'method':
		case 'scope':
		case 'targetType':
		case 'via':
			return enumerated(key, value);
		case 'role':
			return change(value, role) ?? role(value);
		case 'targetRole':
			return role(value);
		case 'active':
			return change(value, bool) ?? bool(value);
		case 'generatedPassword':
			return bool(value);
		case 'changed':
			return changedFields(event, value);
		default:
			if (IDENTIFIER_KEYS.has(key)) return number(value, false);
			if (COUNT_KEYS.has(key)) return number(value, true);
			return raw(value);
	}
}

/** The details of an entry as label/value rows, in the order the server sent them. */
export function detailRows(entry: Pick<AuditEntry, 'event' | 'details'>): DetailRow[] {
	return Object.entries(entry.details ?? {}).map(([key, value]) => ({
		key,
		label: lookup(`audit.details.keys.${key}`, key),
		value: detailValue(entry.event, key, value)
	}));
}
