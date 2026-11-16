<script lang="ts">
	import type { Snippet } from 'svelte';

	let { label, children }: { label: string; children: Snippet } = $props();
</script>

<div class="grid gap-1 py-4 text-sm/6 sm:grid-cols-[8rem_minmax(0,1fr)] sm:gap-4">
	<dt class="text-ink-muted">{label}</dt>
	<dd class="font-medium wrap-anywhere">{@render children()}</dd>
</div>
