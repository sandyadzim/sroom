<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ArrowLeft } from '@lucide/svelte';
	import CarUnitForm from '#lib/components/admin/CarUnitForm.svelte';
	import Link from '#lib/components/ui/Link.svelte';

	const carId = $derived(page.url.searchParams.get('car') ?? '');
	const listHref = $derived(
		`${resolve('/admin/(staff)/units')}${carId ? `?car=${encodeURIComponent(carId)}` : ''}`
	);
</script>

<svelte:head><title>Create unit — SROOM</title></svelte:head>

<Link href={listHref} class="text-sm font-medium text-ink-muted hover:text-ink">
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" /> Back to physical units
</Link>
<h1 class="mt-2 text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Create unit</h1>
{#key carId}<CarUnitForm initialCarId={carId} />{/key}
