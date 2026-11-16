<script lang="ts">
	import CarUnitPage from '#lib/components/admin/CarUnitPage.svelte';
	import type { PageProps } from './$types';

	let { params }: PageProps = $props();
</script>

<CarUnitPage id={params.id} editing />
