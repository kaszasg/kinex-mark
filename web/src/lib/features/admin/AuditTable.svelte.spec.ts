// Audit table and details popover: translated events, result words, raw unknown codes.
import { render, screen, within } from '@testing-library/svelte';
import { describe, expect, test } from 'vitest';
import { fmt, t } from '$lib/i18n/index.js';
import AuditDetails from './AuditDetails.svelte';
import AuditTable from './AuditTable.svelte';
import { findFloating, makeAuditEntry, user } from './test-helpers.js';

const row = (id: number) => document.querySelector(`[data-entry-id="${id}"]`) as HTMLElement;

describe('AuditTable', () => {
	test('a successful entry with target and details', async () => {
		const entry = makeAuditEntry({
			event: 'user.updated',
			target: { type: 'user', id: 5, name: 'peter' },
			details: { role: { from: 'developer', to: 'admin' } }
		});
		render(AuditTable, { entries: [entry] });
		expect(screen.getByRole('table', { name: t('audit.table.caption') })).toBeInTheDocument();
		const cells = within(row(entry.id));
		expect(cells.getByText(t('audit.events.user.updated'))).toBeInTheDocument();
		expect(cells.getByText('user.updated')).toBeInTheDocument();
		expect(cells.getByText(t('audit.result.success'))).toBeInTheDocument();
		expect(cells.getByText('kanna')).toBeInTheDocument();
		expect(cells.getByText(t('audit.targets.user'))).toBeInTheDocument();
		expect(cells.getByText('peter')).toBeInTheDocument();
		expect(cells.getByText('192.0.2.10')).toBeInTheDocument();
		const time = row(entry.id).querySelector('time');
		expect(time).toHaveAttribute('datetime', entry.at);
		expect(time?.textContent?.trim()).toBe(fmt.dateTime(entry.at));
		await user().click(cells.getByRole('button', { name: t('audit.table.showDetails') }));
		const term = await findFloating('term', t('audit.details.keys.role'));
		expect(term.nextElementSibling).toHaveTextContent(
			`${t('roles.developer')} → ${t('roles.admin')}`
		);
	});

	test('a failure without actor, target, IP or details shows dashes', () => {
		const entry = makeAuditEntry({
			event: 'auth.login.failure',
			success: false,
			actor: { id: null, username: '' },
			ip: ''
		});
		render(AuditTable, { entries: [entry] });
		const cells = within(row(entry.id));
		expect(cells.getByText(t('audit.result.failure'))).toBeInTheDocument();
		expect(cells.getAllByText(fmt.missing)).toHaveLength(4);
		expect(cells.queryByRole('button')).toBeNull();
	});

	test('unknown event and target codes stay raw without a code line', () => {
		const entry = makeAuditEntry({
			event: 'robot.dance',
			target: { type: 'planet' as never, id: null, name: '' }
		});
		render(AuditTable, { entries: [entry] });
		const cells = within(row(entry.id));
		expect(cells.getAllByText('robot.dance')).toHaveLength(1);
		expect(cells.getByText('planet')).toHaveClass('font-mono');
	});
});

describe('AuditDetails', () => {
	test('lists every detail with translated and raw values', async () => {
		render(AuditDetails, {
			entry: { event: 'auth.login.failure', details: { reason: 'credentials', extra: 'x' } }
		});
		await user().click(screen.getByRole('button', { name: t('audit.table.showDetails') }));
		const reason = await findFloating('term', t('audit.details.keys.reason'));
		expect(reason.nextElementSibling).toHaveTextContent(
			t('audit.details.values.reason.credentials')
		);
		const extra = await findFloating('term', 'extra');
		expect(extra.nextElementSibling).toHaveClass('text-muted-foreground');
	});

	test('no details: a dash and no button', () => {
		render(AuditDetails, { entry: { event: 'auth.logout', details: {} } });
		expect(screen.getByText(fmt.missing)).toBeInTheDocument();
		expect(screen.queryByRole('button')).toBeNull();
	});
});
