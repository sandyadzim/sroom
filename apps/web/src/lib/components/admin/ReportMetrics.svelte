<script lang="ts">
	let { items }: { items: { label: string; value: string | number }[] } = $props();
</script>

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
		</div>
	{/each}
</dl>
