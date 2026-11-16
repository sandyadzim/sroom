<script lang="ts">
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<div class="mt-6 flex flex-wrap items-center justify-end gap-2 border-t border-line pt-4">
	{@render children()}
</div>
