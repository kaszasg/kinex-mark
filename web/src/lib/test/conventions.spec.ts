// Source conventions of plan 9.8, checked over src/ (generated ui, app.css and messages excluded).
import { readdirSync, readFileSync } from 'node:fs';
import { join, relative, resolve, sep } from 'node:path';
import { parse } from 'svelte/compiler';
import ts from 'typescript';
import { describe, expect, test } from 'vitest';
import { catalogue, LOCALES } from '$lib/i18n/index.js';
import { isPlural } from '$lib/i18n/plural.js';

const SRC = resolve(process.cwd(), 'src');
const SELF = 'lib/test/conventions.spec.ts';

/** Paths relative to src/, with forward slashes. */
function walk(dir: string, out: string[] = []): string[] {
	for (const entry of readdirSync(dir, { withFileTypes: true })) {
		const full = join(dir, entry.name);
		if (entry.isDirectory()) walk(full, out);
		else out.push(relative(SRC, full).split(sep).join('/'));
	}
	return out;
}

function excluded(path: string): boolean {
	return (
		path === SELF ||
		path.startsWith('lib/components/ui/') ||
		path.startsWith('lib/i18n/messages/') ||
		path === 'app.css'
	);
}

const files = walk(SRC).filter((path) => !excluded(path));
const sources = files.filter((path) => /\.(svelte|ts)$/.test(path));
const read = (path: string) => readFileSync(join(SRC, path), 'utf8');

/** Words that stay untranslated everywhere (plan 9.8 rule 5). */
const ALLOWED_WORDS = ['Kinex Mark', 'Kinexium', 'Markdown', 'URL', 'JSON', 'HTML', '2FA'];
const LETTER = /\p{L}/u;

function hasUntranslatedText(text: string): boolean {
	let rest = text;
	for (const word of ALLOWED_WORDS) rest = rest.split(word).join('');
	return LETTER.test(rest);
}

const TEXT_ATTRIBUTES = new Set(['aria-label', 'aria-description', 'title', 'placeholder', 'alt']);

type SvelteNode = { type?: string; [key: string]: unknown };
type AttributeNode = SvelteNode & { name: string; value: true | SvelteNode | SvelteNode[] };

function attributeParts(attr: AttributeNode): SvelteNode[] {
	if (attr.value === true) return [];
	return Array.isArray(attr.value) ? attr.value : [attr.value];
}

function staticText(attr: AttributeNode): string {
	return attributeParts(attr)
		.filter((part) => part.type === 'Text')
		.map((part) => String(part.data ?? ''))
		.join('');
}

/** Visits every template node; attributes are handed to `onAttribute` and not descended into. */
function visit(
	node: unknown,
	onNode: (node: SvelteNode) => void,
	onAttribute: (attr: AttributeNode, owner: SvelteNode) => void,
	owner: SvelteNode | null = null
): void {
	if (Array.isArray(node)) {
		for (const child of node) visit(child, onNode, onAttribute, owner);
		return;
	}
	if (typeof node !== 'object' || node === null) return;
	const n = node as SvelteNode;
	if (n.type === 'Attribute' && owner) {
		onAttribute(n as AttributeNode, owner);
		return;
	}
	if (typeof n.type === 'string') onNode(n);
	for (const [key, value] of Object.entries(n)) {
		if (key === 'metadata' || key === 'parent') continue;
		visit(value, onNode, onAttribute, typeof n.type === 'string' ? n : owner);
	}
}

type ParsedSvelte = {
	fragment: unknown;
	instance?: { content: { body: SvelteNode[] } } | null;
	module?: { content: { body: SvelteNode[] } } | null;
};

const parsed = new Map<string, ParsedSvelte>();
function parseSvelte(path: string): ParsedSvelte {
	let ast = parsed.get(path);
	if (!ast) {
		ast = parse(read(path), { modern: true }) as unknown as ParsedSvelte;
		parsed.set(path, ast);
	}
	return ast;
}

/** Local names imported from a module whose source matches the test. */
function importedNames(ast: ParsedSvelte, matches: (source: string) => boolean): Set<string> {
	const names = new Set<string>();
	for (const script of [ast.instance, ast.module]) {
		for (const statement of script?.content.body ?? []) {
			if (statement.type !== 'ImportDeclaration') continue;
			const source = String((statement.source as { value: unknown }).value);
			if (!matches(source)) continue;
			for (const spec of statement.specifiers as { local: { name: string } }[]) {
				names.add(spec.local.name);
			}
		}
	}
	return names;
}

function lucideViolations(ast: ParsedSvelte, path: string): string[] {
	const offending: string[] = [];
	const icons = importedNames(ast, (source) => source.startsWith('@lucide/svelte'));
	if (icons.size === 0) return offending;
	visit(
		ast.fragment,
		(node) => {
			if (node.type !== 'Component' || !icons.has(String(node.name))) return;
			const attrs = (node.attributes as AttributeNode[]) ?? [];
			const stroke = attrs.find((a) => a.type === 'Attribute' && a.name === 'strokeWidth');
			const part = stroke ? attributeParts(stroke)[0] : undefined;
			const value = (part?.expression as { value?: unknown } | undefined)?.value;
			if (value !== 1.5) offending.push(`${path}: <${String(node.name)}>`);
		},
		() => {}
	);
	return offending;
}

function textViolations(ast: ParsedSvelte, path: string): string[] {
	const offending: string[] = [];
	visit(
		ast.fragment,
		(node) => {
			if (node.type === 'Text' && hasUntranslatedText(String(node.data ?? ''))) {
				offending.push(`${path}: "${String(node.data).trim()}"`);
			}
		},
		(attr) => {
			if (TEXT_ATTRIBUTES.has(attr.name) && hasUntranslatedText(staticText(attr))) {
				offending.push(`${path}: ${attr.name}="${staticText(attr)}"`);
			}
		}
	);
	return offending;
}

function scriptBlocks(path: string): string[] {
	const source = read(path);
	if (path.endsWith('.ts')) return [source];
	return [...source.matchAll(/<script[^>]*>([\s\S]*?)<\/script>/g)].map((m) => m[1]);
}

/** Toast calls whose message or description is a string literal with letters. */
function literalToasts(code: string): string[] {
	const found: string[] = [];
	const file = ts.createSourceFile('x.ts', code, ts.ScriptTarget.Latest, true);
	const isLiteral = (node: ts.Node | undefined): node is ts.StringLiteralLike =>
		node !== undefined && (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node));
	const check = (node: ts.Node | undefined) => {
		if (isLiteral(node) && hasUntranslatedText(node.text)) found.push(node.text);
	};
	const visitTs = (node: ts.Node): void => {
		if (ts.isCallExpression(node)) {
			const callee = node.expression;
			const isToast =
				(ts.isIdentifier(callee) && callee.text === 'toast') ||
				(ts.isPropertyAccessExpression(callee) &&
					ts.isIdentifier(callee.expression) &&
					callee.expression.text === 'toast' &&
					['success', 'error', 'warning', 'info', 'message'].includes(callee.name.text));
			if (isToast) {
				check(node.arguments[0]);
				const options = node.arguments[1];
				if (options && ts.isObjectLiteralExpression(options)) {
					for (const prop of options.properties) {
						if (ts.isPropertyAssignment(prop) && prop.name.getText(file) === 'description') {
							check(prop.initializer);
						}
					}
				}
			}
		}
		ts.forEachChild(node, visitTs);
	};
	visitTs(file);
	return found;
}

describe('plan 9.8 conventions', () => {
	test('the scan sees the source tree', () => {
		expect(sources.length).toBeGreaterThan(20);
		expect(sources).toContain('routes/+layout.svelte');
	});

	test('1. no raw hex colours', () => {
		const offending = sources.filter((path) => /#[0-9a-fA-F]{3,8}\b/.test(read(path)));
		expect(offending).toEqual([]);
	});

	test('2. every Lucide icon uses strokeWidth={1.5}', () => {
		const offending = sources
			.filter((p) => p.endsWith('.svelte'))
			.flatMap((path) => lucideViolations(parseSvelte(path), path));
		expect(offending).toEqual([]);
	});

	test('3. translations contain no "!" and no emoji', () => {
		const offending: string[] = [];
		const collect = (node: unknown, key: string, locale: string) => {
			if (typeof node === 'string') {
				if (node.includes('!') || /\p{Extended_Pictographic}/u.test(node)) {
					offending.push(`${locale}:${key}`);
				}
			} else if (isPlural(node) || (typeof node === 'object' && node !== null)) {
				for (const [k, v] of Object.entries(node)) collect(v, key ? `${key}.${k}` : k, locale);
			}
		};
		for (const locale of LOCALES) collect(catalogue(locale), '', locale);
		expect(offending).toEqual([]);
	});

	test('4. no gradients', () => {
		const word = new RegExp('gradi' + 'ent', 'i');
		const offending = files
			.filter((path) => /\.(svelte|ts|css)$/.test(path))
			.filter((path) => word.test(read(path)));
		expect(offending).toEqual([]);
	});

	test('5a. no letter text nodes and no static text attributes in templates', () => {
		const offending = sources
			.filter((p) => p.endsWith('.svelte'))
			.flatMap((path) => textViolations(parseSvelte(path), path));
		expect(offending).toEqual([]);
	});

	test('5b. no literal toast texts', () => {
		const offending = sources.flatMap((path) =>
			scriptBlocks(path).flatMap((code) => literalToasts(code).map((text) => `${path}: ${text}`))
		);
		expect(offending).toEqual([]);
	});

	test('6. at most one primary button per route and its direct feature components', () => {
		const primaryButtons = (path: string): number => {
			const source = read(path);
			if (source.includes('primary-exception:')) return 0;
			const ast = parseSvelte(path);
			const buttons = importedNames(ast, (s) => s.includes('components/ui/button'));
			let count = 0;
			visit(
				ast.fragment,
				(node) => {
					if (node.type !== 'Component' || !buttons.has(String(node.name))) return;
					const attrs = (node.attributes as AttributeNode[]) ?? [];
					const variant = attrs.find((a) => a.type === 'Attribute' && a.name === 'variant');
					if (!variant || staticText(variant) === 'default') count += 1;
				},
				() => {}
			);
			return count;
		};
		const offending: string[] = [];
		for (const route of sources.filter((p) =>
			/^routes\/.*\+(page|layout|error)\.svelte$/.test(p)
		)) {
			const features = [...read(route).matchAll(/from '\$lib\/(features\/[^']+\.svelte)'/g)].map(
				(m) => `lib/${m[1]}`
			);
			const total = [route, ...features]
				.filter((path) => sources.includes(path))
				.reduce((sum, path) => sum + primaryButtons(path), 0);
			if (total > 1) offending.push(`${route}: ${total}`);
		}
		expect(offending).toEqual([]);
	});

	test('7. no Hungarian or German letters outside the catalogues', () => {
		const allowed = (path: string) =>
			/\.(spec|test)\.ts$/.test(path) || path === 'lib/test/fixtures.ts';
		const offending = sources
			.filter((path) => !allowed(path))
			.filter((path) => /[áéíóöőúüűÁÉÍÓÖŐÚÜŰßäÄ]/.test(read(path)));
		expect(offending).toEqual([]);
	});

	test('8. no dark: Tailwind variant', () => {
		const variant = new RegExp('\\bdark' + ':');
		const offending = sources.filter((path) => variant.test(read(path)));
		expect(offending).toEqual([]);
	});

	test('10. every ToggleGroup item marks its selected state (GRAPHICS.md 6.2, WCAG 1.4.11)', () => {
		// The nova "on" fill alone is about 1.2:1 against the dark page: items use the toggleItem
		// marker of styles.ts or an explicit ring border.
		const offending: string[] = [];
		for (const path of sources.filter((p) => p.endsWith('.svelte'))) {
			for (const match of read(path).matchAll(/<ToggleGroup\.Item\s[^>]*>/g)) {
				if (!/toggleItem|data-\[state=on\]:border-ring/i.test(match[0])) offending.push(path);
			}
		}
		expect(offending).toEqual([]);
	});

	test('11. generated components take the type scale as utilities, not app.css component classes', () => {
		// app.css puts the type styles in @layer components, so the nova utilities on the same
		// element (text-base, leading-none, text-sm, text-xs) win: styles.ts has utility lists.
		const typeClass = /^(heading-[123]|body|body-sm|body-lg|kx-eyebrow|label|code|data|data-lg)$/;
		const offending: string[] = [];
		for (const path of sources.filter((p) => p.endsWith('.svelte'))) {
			const text = read(path);
			const generated = new Set<string>();
			for (const m of text.matchAll(
				/import\s+(?:\*\s+as\s+(\w+)|\{([^}]*)\})\s+from\s+'\$lib\/components\/ui\//g
			)) {
				if (m[1]) generated.add(m[1]);
				for (const name of (m[2] ?? '').split(',')) {
					const local = name
						.split(/\s+as\s+/)
						.pop()
						?.trim();
					if (local) generated.add(local);
				}
			}
			for (const m of text.matchAll(/<([A-Z]\w*)(?:\.\w+)?\s[^>]*?\bclass="([^"]*)"/g)) {
				if (!generated.has(m[1])) continue;
				const hits = m[2].split(/\s+/).filter((token) => typeClass.test(token));
				if (hits.length > 0) offending.push(`${path}: ${hits.join(' ')}`);
			}
		}
		expect(offending).toEqual([]);
	});

	test('9. no cookie besides the session cookie (plan D9, 6.14)', () => {
		// The generated Sidebar.Provider writes a persistent "sidebar_state" cookie; SidebarFrame
		// provides the sidebar context instead.
		const cookie = new RegExp(
			'document\\s*\\.\\s*cookie|cookieStore|Sidebar\\.Provider|SidebarProvider'
		);
		const offending = sources
			.filter((path) => !/\.(spec|test)\.ts$/.test(path))
			.filter((path) => cookie.test(read(path)));
		expect(offending).toEqual([]);
	});
});

describe('phase E conventions', () => {
	const svelteSources = () => sources.filter((p) => p.endsWith('.svelte'));

	test('12. the dialog and sheet overlays are a plain colour scrim, no blur (GRAPHICS.md 10.8)', () => {
		for (const file of [
			'lib/components/ui/dialog/dialog-overlay.svelte',
			'lib/components/ui/alert-dialog/alert-dialog-overlay.svelte',
			'lib/components/ui/sheet/sheet-overlay.svelte'
		]) {
			const text = read(file);
			expect(text, file).toContain('bg-[color-mix(in_oklab,var(--ink-950)_62%,transparent)]');
			expect(text, file).not.toMatch(/blur/);
		}
	});

	test('13. every Dialog.Content and AlertDialog.Content uses dialogBox (user text never widens it)', () => {
		const offending: string[] = [];
		for (const path of svelteSources()) {
			for (const m of read(path).matchAll(/<(?:Alert)?Dialog\.Content\b[^>]*>/g)) {
				if (!/class=\{(?:cn\()?dialogBox\b/.test(m[0])) offending.push(path);
			}
		}
		expect(offending).toEqual([]);
	});

	test('14. a single-select ToggleGroup binds its value through a guard, so it never empties', () => {
		// bits-ui reports '' when the selected item is clicked again; a plain value= with
		// onValueChange lets the group show no selection.
		const offending: string[] = [];
		for (const path of svelteSources()) {
			for (const m of read(path).matchAll(/<ToggleGroup\.Root\b[\s\S]*?>/g)) {
				if (!/type="single"/.test(m[0])) continue;
				if (!/bind:value=\{/.test(m[0]) || /onValueChange/.test(m[0])) offending.push(path);
			}
		}
		expect(offending).toEqual([]);
	});

	test('15. error toasts of failed requests go through toastError (no toast after a lost session)', () => {
		// Client-side checks (no URL found, file too large) are not request failures.
		const direct = /toast\.error\(\s*(?:errorMessage\(\s*(?:err\b|toApiError)|captureError\()/;
		const offending = sources
			.filter((path) => !/\.(spec|test)\.ts$/.test(path) && path !== 'lib/utils/toast-error.ts')
			.filter((path) => direct.test(read(path)));
		expect(offending).toEqual([]);
	});

	test('16. every Checkbox takes checkboxBox (GRAPHICS.md 10.5)', () => {
		const offending: string[] = [];
		for (const path of svelteSources()) {
			for (const m of read(path).matchAll(/<Checkbox\b[\s\S]*?\/>/g)) {
				if (!/checkboxBox/.test(m[0])) offending.push(path);
			}
		}
		expect(offending).toEqual([]);
	});

	test('17. a form with a current-password field names the account for password managers', () => {
		const offending = svelteSources()
			.filter((path) => /autocomplete="current-password"/.test(read(path)))
			// The sign-in form has its own visible username field; the user dialog's field is the
			// admin's step-up password inside a form about another account.
			.filter(
				(path) =>
					path !== 'lib/features/auth/LoginForm.svelte' &&
					path !== 'lib/features/admin/UserDialog.svelte'
			)
			.filter((path) => !/<AccountUsername\b/.test(read(path)));
		expect(offending).toEqual([]);
	});
});

describe('the checks catch violations', () => {
	test('template checks find literal text and missing stroke widths', () => {
		const sample = [
			'<scr' + 'ipt>import X from "@lucide/svelte/icons/x";</scr' + 'ipt>',
			'<X /><X strokeWidth={1.5} /><X strokeWidth={2} />',
			'<p>Hello</p><p>{label}</p><img alt="Logo" src="a.png" /><b title="{a}: {b}"></b>'
		].join('');
		const ast = parse(sample, { modern: true }) as unknown as ParsedSvelte;
		expect(lucideViolations(ast, 's')).toEqual(['s: <X>', 's: <X>']);
		expect(textViolations(ast, 's')).toEqual(['s: "Hello"', 's: alt="Logo"']);
	});

	test('text detection ignores the allowed words', () => {
		expect(hasUntranslatedText('Kinex Mark')).toBe(false);
		expect(hasUntranslatedText(' / 2FA ')).toBe(false);
		expect(hasUntranslatedText('Save')).toBe(true);
	});

	test('literal toasts are found, translated ones are not', () => {
		expect(literalToasts("toast.error('Saved'); toast(t('x')); toast('Kinex Mark');")).toEqual([
			'Saved'
		]);
		expect(literalToasts("toast.success(t('a'), { description: 'Details' })")).toEqual(['Details']);
		expect(literalToasts("other.error('Text')")).toEqual([]);
	});
});
