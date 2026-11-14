<script lang="ts">
	import AdminAccountPage from '#lib/components/admin/AdminAccountPage.svelte';
	import type { PageProps } from './$types';

	let { params }: PageProps = $props();
</script>

<AdminAccountPage id={params.id} mode="detail" />
