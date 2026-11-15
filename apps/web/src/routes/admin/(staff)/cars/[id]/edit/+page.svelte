<script lang="ts">
	import CarPage from '#lib/components/admin/CarPage.svelte';
	import type { PageProps } from './$types';

	let { params }: PageProps = $props();
</script>

<CarPage id={params.id} editing />
