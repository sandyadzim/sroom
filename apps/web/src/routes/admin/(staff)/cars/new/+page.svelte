<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';
	import CarForm from '#lib/components/admin/CarForm.svelte';
	import Link from '#lib/components/ui/Link.svelte';
</script>

<svelte:head><title>Create car — SROOM</title></svelte:head>

<Link
	href={resolve('/admin/(staff)/cars')}
	class="text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" /> Back to cars
</Link>
<h1 class="mt-2 text-2xl/8 font-bold tracking-tight sm:text-3xl/10">Create car</h1>
<CarForm />
