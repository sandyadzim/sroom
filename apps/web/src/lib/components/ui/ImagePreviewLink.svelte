<script lang="ts">
	import ImagePreviewDialog from './ImagePreviewDialog.svelte';

	let {
		title,
		label,
		src,
		alt
	}: {
		title: string;
		label: string;
		src?: string | null;
		alt?: string;
	} = $props();

	const characters = $derived(Array.from(label));
	const suffix = $derived(characters.length > 22 ? characters.slice(-10).join('') : '');
</script>

<ImagePreviewDialog {title} filename={alt || label} {src}>
	{#snippet trigger({ props })}
		<button
			{...props}
			type="button"
			title={label}
			aria-label={label}
			class="flex max-w-40 min-w-0 cursor-pointer rounded-sm text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary sm:max-w-48"
		>
			<span class="truncate">{suffix ? characters.slice(0, -10).join('') : label}</span>
			{#if suffix}<span class="shrink-0 whitespace-pre">{suffix}</span>{/if}
		</button>
	{/snippet}
</ImagePreviewDialog>
