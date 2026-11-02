<script lang="ts">
	import { ChevronLeft, ChevronRight } from '@lucide/svelte';
	import Button from './Button.svelte';

	let {
		page,
		totalPages,
		onchange
	}: { page: number; totalPages: number; onchange: (page: number) => void } = $props();
</script>

<nav aria-label="Car catalogue pages" class="flex flex-wrap items-center justify-center gap-2">
	<Button
		variant="ghost"
		size="icon"
		aria-label="Previous page"
		disabled={page <= 1}
		onclick={() => onchange(page - 1)}
	>
		<ChevronLeft size={20} strokeWidth={1.5} aria-hidden="true" />
	</Button>
	{#each Array.from({ length: totalPages }, (_, index) => index + 1) as number (number)}
		<Button
			variant={page === number ? 'primary' : 'ghost'}
			size="icon"
			aria-label={`Page ${number}`}
			aria-current={page === number ? 'page' : undefined}
			onclick={() => onchange(number)}>{number}</Button
		>
	{/each}
	<Button
		variant="ghost"
		size="icon"
		aria-label="Next page"
		disabled={page >= totalPages}
		onclick={() => onchange(page + 1)}
	>
		<ChevronRight size={20} strokeWidth={1.5} aria-hidden="true" />
	</Button>
</nav>
