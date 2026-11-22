// Hungarian UI texts for the E2E helpers (plan 9.4.1): the helpers resolve accessible names
// from the real catalogue (msg('links.toolbar.add')), so no accented literal lives below
// support/ (langcheck) and a changed translation cannot silently break a selector.
//
// The catalogue is loaded at module evaluation with a dynamic import: the E2E type check
// does not compile the application sources, and a missing catalogue fails with a clear error.
import { fileURLToPath, pathToFileURL } from 'node:url';

const CATALOGUE = fileURLToPath(
	new URL('../../src/lib/i18n/messages/hu/index.ts', import.meta.url)
);

type Plural = { zero?: string; one: string; other: string };
type CatalogueNode = string | Plural | { [key: string]: CatalogueNode };
export type MessageParams = Record<string, string | number>;

async function loadCatalogue(): Promise<Record<string, CatalogueNode>> {
	let module: { default?: unknown; hu?: unknown };
	try {
		module = (await import(pathToFileURL(CATALOGUE).href)) as { default?: unknown; hu?: unknown };
	} catch (err) {
		throw new Error(
			`cannot load the Hungarian message catalogue ${CATALOGUE}: ${err instanceof Error ? err.message : String(err)}`
		);
	}
	const catalogue = module.default ?? module.hu;
	if (catalogue === null || typeof catalogue !== 'object') {
		throw new Error(`${CATALOGUE} has no default export with the message catalogue`);
	}
	return catalogue as Record<string, CatalogueNode>;
}

export const hu = await loadCatalogue();

const pluralRules = new Intl.PluralRules('hu');
// Same as fmt.number in the app (plan 8.5.2): grouping from five digits, no-break space.
const numberFormat = new Intl.NumberFormat('hu', { useGrouping: 'min2' });

function formatParam(value: string | number): string {
	return typeof value === 'number'
		? numberFormat.format(value).replace(/[\u0020\u202f]/g, '\u00a0')
		: value;
}

function isPlural(node: CatalogueNode): node is Plural {
	return typeof node === 'object' && typeof node.one === 'string' && typeof node.other === 'string';
}

/**
 * Returns the Hungarian text of a message key with {name} placeholders filled in (numbers
 * formatted like fmt.number). A plural
 * message needs a numeric count parameter. Throws on unknown keys.
 */
export function msg(key: string, params: MessageParams = {}): string {
	let node: CatalogueNode | undefined = hu;
	for (const part of key.split('.')) {
		node = typeof node === 'object' && !isPlural(node) ? node[part] : undefined;
		if (node === undefined) throw new Error(`unknown message key: ${key}`);
	}
	let text: string;
	if (typeof node === 'string') {
		text = node;
	} else if (isPlural(node)) {
		const count = params.count;
		if (typeof count !== 'number') throw new Error(`plural message ${key} needs a numeric count`);
		text =
			count === 0 && node.zero !== undefined
				? node.zero
				: node[pluralRules.select(count) === 'one' ? 'one' : 'other'];
	} else {
		throw new Error(`message key ${key} is a namespace, not a message`);
	}
	return text.replace(/\{(\w+)\}/g, (match, name: string) =>
		name in params ? formatParam(params[name]) : match
	);
}
