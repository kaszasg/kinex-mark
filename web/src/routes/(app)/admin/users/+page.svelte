<script lang="ts">
	import PageTitle from '$lib/components/app/PageTitle.svelte';
	import UsersScreen from '$lib/features/admin/UsersScreen.svelte';
	import { t } from '$lib/i18n/index.js';
</script>

<PageTitle title={t('admin.users.pageTitle')} />
<UsersScreen />
