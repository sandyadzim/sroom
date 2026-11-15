<script lang="ts">
	import type { ComponentProps } from 'svelte';
	import Dialog from './Dialog.svelte';
	import ImageWithPlaceholder from './ImageWithPlaceholder.svelte';

	let {
		title,
		filename,
		src,
		trigger
	}: {
		title: string;
		filename: string;
		src?: string | null;
		trigger: ComponentProps<typeof Dialog>['trigger'];
	} = $props();
</script>

<Dialog {title} description="Image preview" closeLabel="Close image preview" {trigger}>
	<ImageWithPlaceholder
		{src}
		alt={filename}
		fit="scale-down"
		class="h-[min(55dvh,30rem)] rounded-lg sm:h-[min(65dvh,38rem)]"
	>
		{#snippet fallback()}
			<p class="px-4 text-center text-sm text-ink-muted">Image preview is unavailable.</p>
		{/snippet}
	</ImageWithPlaceholder>
</Dialog>
