<script lang="ts">
	let {
		items,
		loading = false
	}: {
		items: { label: string; value: string | number; detail?: string }[];
		loading?: boolean;
	} = $props();
	const skeletonCount = 5;
</script>

{#if loading}
	<dl
		aria-busy="true"
		class="grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-line bg-line xl:grid-cols-5"
	>
		{#each [...Array(skeletonCount).keys()] as index (index)}
			<div class="min-w-0 bg-panel px-3 py-4 odd:last:col-span-2 sm:px-5 xl:odd:last:col-span-1">
				<div class="h-4 w-24 rounded bg-panel-muted motion-safe:animate-pulse"></div>
				<div class="mt-3 h-7 w-16 rounded bg-panel-muted motion-safe:animate-pulse"></div>
			</div>
		{/each}
	</dl>
	<p role="status" class="sr-only">Loading report metrics…</p>
{:else}
	<dl
		class="grid grid-cols-2 gap-px overflow-hidden rounded-xl border border-line bg-line xl:grid-cols-5"
	>
		{#each items as item (item.label)}
			<div class="min-w-0 bg-panel px-3 py-4 odd:last:col-span-2 sm:px-5 xl:odd:last:col-span-1">
				<dt class="text-sm leading-5 text-ink-muted">{item.label}</dt>
				<dd
					class="mt-2 font-display text-xl leading-snug font-bold wrap-anywhere tabular-nums sm:text-2xl"
				>
					{item.value}
				</dd>
				{#if item.detail}
					<p class="mt-1 text-xs leading-5 text-ink-muted">{item.detail}</p>
				{/if}
			</div>
		{/each}
	</dl>
{/if}
