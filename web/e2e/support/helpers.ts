// UI helpers of the E2E suite (plan 9.4.1). They take accessible names as parameters, so a
// spec passes texts resolved from the Hungarian catalogue:
//
//   import { msg } from '../support/i18n.ts';
//   await addLink(page, { url }, {
//     open: msg('links.toolbar.add'), url: msg('links.dialog.url'), submit: msg('links.dialog.save')
//   });
//
// No accented literal lives below support/ (langcheck); specs may contain them.
import { expect, type Locator, type Page } from '@playwright/test';

export interface NewLinkInput {
	url: string;
	title?: string;
	description?: string;
}

/** Accessible names of the add-link flow (button, dialog fields, submit button). */
export interface AddLinkNames {
	/** Button that opens the dialog. */
	open: string;
	/** Label of the URL field. */
	url: string;
	/** Label of the title field; required when input.title is set. */
	title?: string;
	/** Label of the description field; required when input.description is set. */
	description?: string;
	/** Submit button of the dialog. */
	submit: string;
}

/**
 * Adds a link through the add-link dialog and waits until the dialog has closed and the
 * POST /api/links response arrived. Returns the created link id when the server created one.
 */
export async function addLink(
	page: Page,
	input: NewLinkInput,
	names: AddLinkNames
): Promise<number | undefined> {
	await page.getByRole('button', { name: names.open, exact: true }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog).toBeVisible();
	await dialog.getByLabel(names.url, { exact: true }).fill(input.url);
	if (input.title !== undefined) {
		if (!names.title) throw new Error('addLink: names.title is required when a title is given');
		await dialog.getByLabel(names.title, { exact: true }).fill(input.title);
	}
	if (input.description !== undefined) {
		if (!names.description) {
			throw new Error('addLink: names.description is required when a description is given');
		}
		await dialog.getByLabel(names.description, { exact: true }).fill(input.description);
	}
	const created = page.waitForResponse(
		(response) =>
			response.request().method() === 'POST' && new URL(response.url()).pathname === '/api/links'
	);
	await dialog.getByRole('button', { name: names.submit, exact: true }).click();
	const response = await created;
	if (response.status() !== 201) return undefined;
	await expect(dialog).toBeHidden();
	const body = (await response.json()) as { id?: number };
	return body.id;
}

/** The list row (li.kxm-list-row, plan 8.7.3) whose content contains the given text. */
export function linkRow(page: Page, text: string | RegExp): Locator {
	return page.locator('li.kxm-list-row').filter({ hasText: text }).first();
}

/**
 * Opens the actions menu of a link row and returns the menu. The menu button is only visible
 * on hover or focus-within for mouse users (plan 8.7.3), so the row is hovered first.
 */
export async function openMenu(page: Page, row: Locator, menuButtonName: string): Promise<Locator> {
	await row.hover();
	await row.getByRole('button', { name: menuButtonName, exact: true }).click();
	const menu = page.getByRole('menu');
	await expect(menu).toBeVisible();
	return menu;
}

export interface DropOptions {
	/** Anchor text placed in text/html, which the app uses as the title. */
	title?: string;
	/** Called after dragenter and dragover, before drop: assert the drop overlay here. */
	beforeDrop?: () => Promise<void>;
}

/**
 * Drags a URL onto the page with a synthetic DataTransfer (text/uri-list and text/html):
 * dragenter, dragover and drop on document.body, like a link dragged from another window.
 */
export async function dropUrl(page: Page, url: string, options: DropOptions = {}): Promise<void> {
	const dataTransfer = await page.evaluateHandle(
		({ url: href, title }) => {
			const transfer = new DataTransfer();
			transfer.setData('text/uri-list', href);
			transfer.setData('text/plain', href);
			if (title !== undefined) {
				const anchor = document.createElement('a');
				anchor.href = href;
				anchor.textContent = title;
				transfer.setData('text/html', anchor.outerHTML);
			}
			return transfer;
		},
		{ url, title: options.title }
	);
	try {
		for (const type of ['dragenter', 'dragover'] as const) {
			await page.dispatchEvent('body', type, { dataTransfer });
		}
		if (options.beforeDrop) await options.beforeDrop();
		await page.dispatchEvent('body', 'drop', { dataTransfer });
	} finally {
		await dataTransfer.dispose();
	}
}

export interface PasteOptions {
	/** Optional text/html content of the clipboard. */
	html?: string;
	/** Element that receives the event (default: document.body, outside any input). */
	target?: Locator;
}

/** Pastes text with a synthetic ClipboardEvent, like Ctrl+V outside an input field. */
export async function pasteText(
	page: Page,
	text: string,
	options: PasteOptions = {}
): Promise<void> {
	const target = options.target ?? page.locator('body');
	await target.evaluate(
		(element, { text: plain, html }) => {
			const transfer = new DataTransfer();
			transfer.setData('text/plain', plain);
			if (html !== undefined) transfer.setData('text/html', html);
			element.dispatchEvent(
				new ClipboardEvent('paste', { clipboardData: transfer, bubbles: true, cancelable: true })
			);
		},
		{ text, html: options.html }
	);
}
