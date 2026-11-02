<script lang="ts">
	import { ArrowUpRight } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import Link from '#lib/components/ui/Link.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import { formatIDR } from '#lib/utils/currency.ts';
	import type { CarPreview } from '#lib/types/car.ts';

	let {
		car,
		bookingHref,
		compact = false
	}: { car: CarPreview; bookingHref: string; compact?: boolean } = $props();
</script>

{#if compact}
	<div
		class="sticky bottom-0 z-30 bg-panel pt-3 pb-[max(0.75rem,env(safe-area-inset-bottom))] lg:hidden"
		aria-label="Quick booking"
	>
		<div class="page-container flex items-center justify-between gap-3">
			<p class="text-lg/7 font-semibold tabular-nums">
				{formatIDR(car.dailyRate)}<span class="block text-sm/5 font-normal text-ink-muted"
					>per day</span
				>
			</p>
			<Button
				href={bookingHref}
				class="shrink-0"
				aria-label={`Book Now — ${car.brand} ${car.model} ${car.trim}`}
				>Book Now <ArrowUpRight size={18} strokeWidth={1.5} aria-hidden="true" /></Button
			>
		</div>
	</div>
{:else}
	<div class="rounded-panel bg-panel-muted p-6 sm:p-8 lg:p-6">
		<h2 class="text-lg/7 font-medium">Your rental</h2>
		<p class="mt-3 text-3xl/10 font-semibold tracking-tight tabular-nums">
			{formatIDR(car.dailyRate)}<span
				class="ml-1 text-sm/6 font-normal tracking-normal text-ink-muted">/ day</span
			>
		</p>
		<p class="mt-3 text-sm/6 text-ink-muted">
			Your final total and availability are confirmed for the dates you choose at checkout.
		</p>
		<Button
			href={bookingHref}
			class="mt-6 w-full"
			aria-label={`Book Now — ${car.brand} ${car.model} ${car.trim}`}
			>Book Now <ArrowUpRight size={18} strokeWidth={1.5} aria-hidden="true" /></Button
		>
		<p class="mt-3 text-center text-sm/6 text-ink-muted">Sign in to continue your booking.</p>
		<Link href={resolve('/rental-terms')} class="mt-3 w-full text-sm text-ink-muted" arrow
			>Rental terms</Link
		>
	</div>
{/if}
