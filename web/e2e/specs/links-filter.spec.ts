// Search and tag filters (plan 9.4.2 links-filter.spec.ts): six links with three tags, text
// search over title, description, URL and tag name, any/all tag matching, untagged, scope, the
// URL as the source of the query, clearing, and grouping by tag from the sidebar and /tags.
import type { Page } from '@playwright/test';
import { test, expect, type CreatedUser } from '../support/fixtures.ts';
import {
	apiCreateLink,
	apiCreateTag,
	exactText,
	fixtureUrl,
	gotoLinks,
	msg,
	rowLabels,
	withApi,
	type ApiTag
} from './lib/app.ts';

interface Collection {
	alpha: ApiTag;
	beta: ApiTag;
	gamma: ApiTag;
}

async function seed(user: CreatedUser, testId: string): Promise<Collection> {
	return withApi(user, async (api) => {
		const alpha = await apiCreateTag(api, 'alpha-tag', 'blue');
		const beta = await apiCreateTag(api, 'beta-tag', 'green');
		const gamma = await apiCreateTag(api, 'gamma-tag', 'violet');
		const links = [
			{ name: 'l1', description: 'Kubernetes runbook', tagIds: [alpha.id], pinned: true },
			{ name: 'l2', description: 'Grafana board', tagIds: [alpha.id, beta.id] },
			{ name: 'l3', description: 'Release notes', tagIds: [beta.id] },
			{ name: 'l4', description: 'Incident form', tagIds: [gamma.id] },
			{ name: 'l5', description: 'Coffee menu', title: 'Lunch place' },
			{ name: 'urltoken', description: 'Team wiki' }
		];
		for (const link of links) {
			await apiCreateLink(api, {
				url: fixtureUrl(testId, link.name),
				title: link.title ?? 'Common title',
				description: link.description,
				tagIds: link.tagIds ?? [],
				pinned: link.pinned ?? false
			});
		}
		return { alpha, beta, gamma };
	});
}

const sorted = async (page: Page) => [...(await rowLabels(page))].sort();

async function search(page: Page, text: string): Promise<void> {
	const box = page.getByRole('searchbox', { name: msg('nav.search.label') });
	await box.fill(text);
	await expect(page).toHaveURL(new RegExp(`[?&]q=${encodeURIComponent(text)}`));
	await expect(page.locator('[aria-busy="true"]')).toHaveCount(0);
}

const filterBar = (page: Page) => page.getByRole('group', { name: msg('links.filters.label') });

/** The number in the sidebar badge next to "Links". */
async function sidebarLinkCount(page: Page): Promise<string> {
	const item = page
		.getByRole('navigation', { name: msg('nav.sidebarTitle') })
		.locator('li')
		.filter({ has: page.getByRole('link', { name: msg('nav.links'), exact: true }) });
	return (await item.innerText()).replace(msg('nav.links'), '').trim();
}

test('text search covers title, description, URL and tag name', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	await seed(user, testInfo.testId);
	const page = await openPage(user.storageState);
	await gotoLinks(page);
	await expect(page.locator('li.kxm-list-row')).toHaveCount(6);

	await search(page, 'Lunch');
	expect(await sorted(page)).toEqual(['Coffee menu']);
	await search(page, 'runbook');
	expect(await sorted(page)).toEqual(['Kubernetes runbook']);
	await search(page, 'urltoken');
	expect(await sorted(page)).toEqual(['Team wiki']);
	await search(page, 'gamma-tag');
	expect(await sorted(page)).toEqual(['Incident form']);
	await expect(page.getByText(msg('links.summary', { count: 1, pinned: 0 }))).toBeVisible();

	await search(page, 'nothing-matches-this');
	await expect(page.getByText(msg('links.empty.filtered.title'))).toBeVisible();
	await expect(
		page.getByRole('paragraph').filter({ hasText: exactText(msg('links.results', { count: 0 })) })
	).toBeVisible();
	await page
		.getByRole('button', { name: msg('links.filters.clear'), exact: true })
		.first()
		.click();
	await expect(page.locator('li.kxm-list-row')).toHaveCount(6);
	await expect(page).not.toHaveURL(/[?&]q=/);
});

test('tag filters match any or all tags, untagged and scope narrow the list', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('developer');
	const tags = await seed(user, testInfo.testId);
	const page = await openPage(user.storageState);
	await gotoLinks(page);

	await filterBar(page).getByRole('button', { name: 'alpha-tag', exact: true }).click();
	await expect.poll(() => sorted(page)).toEqual(['Grafana board', 'Kubernetes runbook']);
	// The subtitle counts the filtered set; the sidebar counter does not follow the filter.
	await expect(page.getByText(msg('links.summary', { count: 2, pinned: 1 }))).toBeVisible();
	expect(await sidebarLinkCount(page)).toBe('6');

	await filterBar(page).getByRole('button', { name: 'beta-tag', exact: true }).click();
	await expect
		.poll(() => sorted(page))
		.toEqual(['Grafana board', 'Kubernetes runbook', 'Release notes']);
	// "All" is also saved as the preferred match mode (debounced PATCH).
	const saved = page.waitForResponse(
		(response) =>
			response.request().method() === 'PATCH' &&
			new URL(response.url()).pathname === '/api/me/preferences'
	);
	await page
		.getByRole('group', { name: msg('links.filters.match.label') })
		.getByRole('radio', { name: msg('links.filters.match.all'), exact: true })
		.click();
	expect((await saved).status()).toBe(200);
	await expect.poll(() => sorted(page)).toEqual(['Grafana board']);
	await expect(page).toHaveURL(new RegExp(`tags=${tags.alpha.id}(%2C|,)${tags.beta.id}`));
	await expect(
		page.getByRole('group', { name: msg('links.filters.match.label') }).getByRole('radio', {
			name: msg('links.filters.match.all'),
			exact: true
		})
	).toHaveAttribute('aria-checked', 'true');

	// The URL is the query: a reload keeps it.
	await page.reload();
	await expect.poll(() => sorted(page)).toEqual(['Grafana board']);
	await expect(
		filterBar(page).getByRole('button', { name: 'alpha-tag', exact: true })
	).toHaveAttribute('aria-pressed', 'true');

	await filterBar(page)
		.getByRole('button', { name: msg('links.filters.untagged'), exact: true })
		.click();
	await expect.poll(() => sorted(page)).toEqual(['Coffee menu', 'Team wiki']);

	await filterBar(page)
		.getByRole('button', { name: msg('links.filters.clear'), exact: true })
		.click();
	await expect(page.locator('li.kxm-list-row')).toHaveCount(6);

	const scope = page.getByRole('group', { name: msg('links.toolbar.scope') });
	await scope.getByRole('radio', { name: msg('links.scope.shared'), exact: true }).click();
	await expect(page.getByText(msg('links.empty.filtered.title'))).toBeVisible();
	await scope.getByRole('radio', { name: msg('links.scope.own'), exact: true }).click();
	await expect(page.locator('li.kxm-list-row')).toHaveCount(6);
	await page.reload();
	await expect(
		scope.getByRole('radio', { name: msg('links.scope.own'), exact: true })
	).toHaveAttribute('aria-checked', 'true');
});

test('the sidebar tag and the tag count link group the links of one tag', async ({
	createUser,
	openPage
}, testInfo) => {
	const user = await createUser('customer');
	await seed(user, testInfo.testId);
	const page = await openPage(user.storageState);
	await gotoLinks(page);

	const sidebar = page.getByRole('navigation', { name: msg('nav.sidebarTitle') });
	await expect(sidebar.getByText(msg('nav.groups.frequentTags'))).toBeVisible();
	await sidebar.getByRole('link', { name: 'alpha-tag' }).click();
	await expect.poll(() => sorted(page)).toEqual(['Grafana board', 'Kubernetes runbook']);

	await sidebar.getByRole('link', { name: msg('nav.tags'), exact: true }).click();
	await expect(page.getByRole('heading', { level: 1, name: msg('tags.pageTitle') })).toBeVisible();
	await page
		.getByRole('link', { name: msg('tags.table.showLinks', { count: 2, name: 'beta-tag' }) })
		.click();
	await expect(page).toHaveURL(/\/\?tags=\d+/);
	// The link with two tags shows in both groups.
	await expect.poll(() => sorted(page)).toEqual(['Grafana board', 'Release notes']);
	await expect(page.getByText(msg('links.summary', { count: 2, pinned: 0 }))).toBeVisible();
});
