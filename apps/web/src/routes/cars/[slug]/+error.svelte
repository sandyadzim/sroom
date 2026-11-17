<script lang="ts">
	import { page } from '$app/state';
	import { refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Button from '#lib/components/ui/Button.svelte';

	let retrying = $state(false);

	async function retry() {
		retrying = true;
		try {
			await refreshAll();
		} finally {
			retrying = false;
		}
	}
</script>

<svelte:head
	><title>{page.status === 404 ? 'Car not found' : 'Unable to load car'} — SROOM</title><meta
		name="robots"
		content="noindex"
	/></svelte:head
>
<section class="page-container py-16 sm:py-24" aria-labelledby="car-error-heading">
	<h1 id="car-error-heading" class="text-3xl/tight font-semibold tracking-tight sm:text-4xl/tight">
		{page.status === 404 ? 'Car not found.' : 'Unable to load car.'}
	</h1>
	<p class="mt-4 max-w-lg text-base/7 text-ink-muted">
		{page.status === 404
			? 'The car you’re looking for could not be found. Explore the collection to find your next drive.'
			: 'Please try again, or explore the collection for another option.'}
	</p>
	<div class="mt-6 flex flex-wrap gap-3">
		{#if page.status !== 404}
			<Button variant="secondary" disabled={retrying} onclick={retry}>
				{retrying ? 'Loading…' : 'Try again'}
			</Button>
		{/if}
		<Button href={resolve('/cars')}>Explore Cars</Button>
	</div>
</section>
