// Small browser helpers (plan 8.3, 8.7.0, 8.7.3, 8.7.6).
import { afterEach, describe, expect, test, vi } from 'vitest';
import { toast } from 'svelte-sonner';
import { i18n, t } from '$lib/i18n/index.js';
import { copyText } from './clipboard.js';
import { debounce } from './debounce.js';
import { saveBlob, saveText } from './download.js';
import {
	isApplePlatform,
	isDialogOpen,
	isEditableElement,
	keyLabel,
	modifierKeyLabel,
	shouldHandleShortcut
} from './keyboard.js';
import { loginUrl, safeNext } from './navigation.js';
import { qrMatrix, qrPath, QUIET_ZONE } from './qr.js';
import { cleanQueryText } from './text.js';
import { describeUserAgent, rawUserAgent } from './user-agent.js';

afterEach(() => {
	vi.restoreAllMocks();
	vi.useRealTimers();
	document.body.innerHTML = '';
});

describe('safeNext', () => {
	const origin = 'https://mark.example';
	test.each([
		[null, '/'],
		[undefined, '/'],
		['', '/'],
		['/', '/'],
		['/tags', '/tags'],
		['/?q=a&tags=1#top', '/?q=a&tags=1#top'],
		['/settings/security', '/settings/security'],
		['https://mark.example/tags', '/tags'],
		['https://evil.example/tags', '/'],
		['//evil.example', '/'],
		['/\\evil.example', '/'],
		['\\\\evil.example', '/'],
		['/%2F%2Fevil.example', '/'],
		['/%5Cevil.example', '/'],
		['/%E0%A4%A', '/'],
		['javascript:alert(1)', '/'],
		['/tags\u0000', '/'],
		['/tags\n', '/'],
		['/tags\u007f', '/'],
		['http://[', '/'],
		['relative/path', '/relative/path']
	])('%j -> %s', (next, expected) => {
		expect(safeNext(next, origin)).toBe(expected);
	});

	test('loginUrl keeps the return path and the reason', () => {
		expect(loginUrl('/')).toBe('/login');
		expect(loginUrl('/tags?x=1')).toBe('/login?next=%2Ftags%3Fx%3D1');
		expect(loginUrl('/', 'expired')).toBe('/login?reason=expired');
		expect(loginUrl('/tags', 'deleted')).toBe('/login?next=%2Ftags&reason=deleted');
		expect(loginUrl('')).toBe('/login');
	});
});

describe('debounce', () => {
	test('runs once with the last arguments', () => {
		vi.useFakeTimers();
		const fn = vi.fn();
		const debounced = debounce(fn, 100);
		debounced(1);
		debounced(2);
		expect(debounced.pending).toBe(true);
		vi.advanceTimersByTime(99);
		expect(fn).not.toHaveBeenCalled();
		vi.advanceTimersByTime(1);
		expect(fn).toHaveBeenCalledOnce();
		expect(fn).toHaveBeenCalledWith(2);
		expect(debounced.pending).toBe(false);
	});

	test('cancel drops the pending call and flush runs it now', () => {
		vi.useFakeTimers();
		const fn = vi.fn();
		const debounced = debounce(fn, 100);
		debounced('a');
		debounced.cancel();
		vi.advanceTimersByTime(200);
		expect(fn).not.toHaveBeenCalled();
		debounced.cancel();
		debounced.flush();
		expect(fn).not.toHaveBeenCalled();
		debounced('b');
		debounced.flush();
		expect(fn).toHaveBeenCalledWith('b');
	});
});

describe('saveBlob', () => {
	test('clicks a temporary download link and revokes the URL', () => {
		vi.useFakeTimers();
		const create = vi.fn(() => 'blob:x');
		const revoke = vi.fn();
		Object.assign(URL, { createObjectURL: create, revokeObjectURL: revoke });
		const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
		saveText('codes', 'recovery-codes.txt');
		expect(click).toHaveBeenCalledOnce();
		const anchor = click.mock.contexts[0] as HTMLAnchorElement;
		expect(anchor.download).toBe('recovery-codes.txt');
		expect(document.querySelector('a')).toBeNull();
		vi.runAllTimers();
		expect(revoke).toHaveBeenCalledWith('blob:x');
		saveBlob(new Blob(['x']), 'a.json');
		expect(click).toHaveBeenCalledTimes(2);
	});
});

describe('copyText', () => {
	test('uses the async clipboard', async () => {
		await expect(copyText('hello')).resolves.toBe(true);
		await expect(navigator.clipboard.readText()).resolves.toBe('hello');
	});

	test('falls back to execCommand when the clipboard rejects', async () => {
		vi.spyOn(navigator.clipboard, 'writeText').mockRejectedValueOnce(new Error('denied'));
		const exec = vi.fn(() => true);
		Object.assign(document, { execCommand: exec });
		await expect(copyText('x')).resolves.toBe(true);
		expect(exec).toHaveBeenCalledWith('copy');
		expect(document.querySelector('textarea')).toBeNull();
	});

	test('shows an error toast when every way fails', async () => {
		const error = vi.spyOn(toast, 'error').mockImplementation(() => 1);
		const clipboard = navigator.clipboard;
		Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined });
		Object.assign(document, {
			execCommand: () => {
				throw new Error('unsupported');
			}
		});
		try {
			await expect(copyText('x')).resolves.toBe(false);
			expect(error).toHaveBeenCalledWith(t('common.copyFailed'));
		} finally {
			Object.defineProperty(navigator, 'clipboard', { configurable: true, value: clipboard });
		}
	});
});

describe('keyboard', () => {
	function key(init: KeyboardEventInit & { target?: EventTarget }): KeyboardEvent {
		const event = new KeyboardEvent('keydown', {
			key: '?',
			bubbles: true,
			cancelable: true,
			...init
		});
		if (init.target) Object.defineProperty(event, 'target', { value: init.target });
		return event;
	}

	test('editable elements block shortcuts', () => {
		document.body.innerHTML = `
			<input id="text" /><input id="check" type="checkbox" /><textarea id="area"></textarea>
			<select id="sel"></select><div id="edit" contenteditable="true"><span id="inner"></span></div>
			<button id="btn"></button>`;
		const byId = (id: string) => document.getElementById(id);
		expect(isEditableElement(byId('text'))).toBe(true);
		expect(isEditableElement(byId('check'))).toBe(false);
		expect(isEditableElement(byId('area'))).toBe(true);
		expect(isEditableElement(byId('sel'))).toBe(true);
		expect(isEditableElement(byId('inner'))).toBe(true);
		expect(isEditableElement(byId('btn'))).toBe(false);
		expect(isEditableElement(null)).toBe(false);
		expect(isEditableElement(window)).toBe(false);
	});

	test('shouldHandleShortcut checks modifiers, focus and dialogs', () => {
		document.body.innerHTML = '<input id="text" /><button id="btn"></button>';
		const btn = document.getElementById('btn')!;
		expect(shouldHandleShortcut(key({ target: btn }))).toBe(true);
		expect(shouldHandleShortcut(key({ target: btn, ctrlKey: true }))).toBe(false);
		expect(shouldHandleShortcut(key({ target: btn, metaKey: true }))).toBe(false);
		expect(shouldHandleShortcut(key({ target: btn, altKey: true }))).toBe(false);
		expect(shouldHandleShortcut(key({ target: document.getElementById('text')! }))).toBe(false);
		const prevented = key({ target: btn });
		prevented.preventDefault();
		expect(shouldHandleShortcut(prevented)).toBe(false);
		document.body.insertAdjacentHTML('beforeend', '<div role="dialog"></div>');
		expect(isDialogOpen()).toBe(true);
		expect(shouldHandleShortcut(key({ target: btn }))).toBe(false);
	});

	test('platform modifier label', () => {
		const platform = vi.spyOn(navigator, 'platform', 'get');
		platform.mockReturnValue('MacIntel');
		expect(isApplePlatform()).toBe(true);
		expect(modifierKeyLabel()).toBe('⌘');
		platform.mockReturnValue('Win32');
		expect(isApplePlatform()).toBe(false);
		expect(modifierKeyLabel()).toBe('Ctrl');
	});

	test('key labels follow the language of the keyboard (German Strg and Entf)', () => {
		vi.spyOn(navigator, 'platform', 'get').mockReturnValue('Win32');
		try {
			i18n.setLocale('de', { persist: false });
			expect(modifierKeyLabel()).toBe('Strg');
			expect(['Enter', 'Delete', 'Esc', 'J', '?'].map(keyLabel)).toEqual([
				'Enter',
				'Entf',
				'Esc',
				'J',
				'?'
			]);
			i18n.setLocale('en', { persist: false });
			expect(modifierKeyLabel()).toBe('Ctrl');
			expect(['Enter', 'Delete', 'Esc'].map(keyLabel)).toEqual(['Enter', 'Delete', 'Esc']);
		} finally {
			i18n.setLocale('hu', { persist: false });
		}
		expect(keyLabel('Delete')).toBe(t('common.keys.delete'));
	});
});

describe('describeUserAgent', () => {
	test.each([
		[
			'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36',
			'Chrome, Windows'
		],
		[
			'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36 Edg/130.0',
			'Edge, Windows'
		],
		['Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0', 'Firefox, Linux'],
		[
			'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15',
			'Safari, macOS'
		],
		[
			'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1',
			'Safari, iOS'
		],
		[
			'Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Mobile Safari/537.36',
			'Chrome, Android'
		],
		['curl/8.5.0', null],
		['SomeBot (Windows)', 'Windows'],
		['Firefox/1.0', 'Firefox']
	])('%s', (ua, expected) => {
		expect(describeUserAgent(ua)).toBe(expected);
	});

	test('raw fallback is cut to 40 characters', () => {
		expect(rawUserAgent('x'.repeat(60))).toHaveLength(40);
	});
});

describe('qr', () => {
	test('matrix has the quiet zone and the path draws dark modules', () => {
		const matrix = qrMatrix('otpauth://totp/Kinex%20Mark:kanna?secret=JBSWY3DPEHPK3PXP');
		expect(matrix.size).toBe(matrix.modules.length);
		expect(matrix.modules[0].every((dark) => !dark)).toBe(true);
		expect(matrix.modules[QUIET_ZONE][QUIET_ZONE]).toBe(true);
		const path = qrPath(matrix);
		expect(path.startsWith(`M${QUIET_ZONE} ${QUIET_ZONE}h1v1h-1z`)).toBe(true);
		expect(qrPath({ size: 1, modules: [[false]] })).toBe('');
	});
});

describe('cleanQueryText', () => {
	test.each([
		['  grafana  ', 10, 'grafana'],
		['a\tb', 10, 'a b'],
		['line\none\r\n', 20, 'line one'],
		['x\u0000y\u007fz\u0085', 10, 'x y z'],
		['\u202eevil\u202c \u2066name\u2069\u200e', 20, 'evil name'],
		// U+061C ARABIC LETTER MARK is a Bidi_Control rune too, as on the server.
		['adm\u061cin', 10, 'admin'],
		['abcdef', 3, 'abc'],
		['ab cd', 3, 'ab'],
		// Runes, not UTF-16 units: an astral character counts once.
		['\u{1d11e}\u{1d11e}\u{1d11e}x', 3, '\u{1d11e}\u{1d11e}\u{1d11e}'],
		['\t \n', 5, '']
	])('cleanQueryText(%j, %i) is %j', (text, max, expected) => {
		expect(cleanQueryText(text, max)).toBe(expected);
	});
});
