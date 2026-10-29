<script lang="ts">
	import type { Snippet } from 'svelte';
	let { children }: { children: Snippet } = $props();
</script>

<div class="page-container flex min-h-24 items-center justify-between gap-6">
	{@render children()}
</div>
