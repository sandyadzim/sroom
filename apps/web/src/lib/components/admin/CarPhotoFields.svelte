<script lang="ts">
	import { Eye, Trash2 } from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import ImagePreviewDialog from '#lib/components/ui/ImagePreviewDialog.svelte';
	import ImageWithPlaceholder from '#lib/components/ui/ImageWithPlaceholder.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import type { CarPhotoInput } from '#lib/types/catalog.ts';

	let {
		photo,
		src,
		title,
		submitted,
		disabled,
		onedit,
		onremove
	}: {
		photo: CarPhotoInput;
		src: string | null;
		title: string;
		submitted: boolean;
		disabled: boolean;
		onedit: (field: 'label' | 'alt_text', value: string) => void;
		onremove?: () => void;
	} = $props();

	let labelTouched = $state(false);
	let altTouched = $state(false);
</script>

<div class="grid min-w-0 gap-3 sm:grid-cols-[8rem_minmax(0,1fr)]">
	<div
		class="group/photo relative h-fit min-w-0 overflow-hidden rounded-xl border border-line bg-panel-muted"
	>
		<ImageWithPlaceholder {src} alt={photo.alt_text} class="aspect-video" />
		<div
			class="absolute inset-x-0 bottom-0 flex justify-end gap-1 bg-panel/90 p-1 pointer-fine:opacity-0 pointer-fine:group-focus-within/photo:opacity-100 pointer-fine:group-hover/photo:opacity-100"
		>
			<ImagePreviewDialog {title} filename={photo.alt_text || title} {src}>
				{#snippet trigger({ props })}
					<Button
						{...props}
						size="icon"
						variant="ghost"
						aria-label={`View ${title.toLowerCase()}`}
						disabled={disabled || !src}
					>
						<Eye size={16} aria-hidden="true" />
					</Button>
				{/snippet}
			</ImagePreviewDialog>
			{#if onremove}
				<Button
					size="icon"
					variant="ghost"
					aria-label={`Remove ${title.toLowerCase()}`}
					{disabled}
					onclick={onremove}
				>
					<Trash2 size={16} aria-hidden="true" />
				</Button>
			{/if}
		</div>
	</div>
	<div class="grid gap-3">
		<TextField
			name="photo-label"
			label="Photo label"
			placeholder="e.g. Front three-quarter view"
			value={photo.label}
			oninput={(event) => onedit('label', event.currentTarget.value)}
			maxlength={100}
			required
			{disabled}
			onblur={() => (labelTouched = true)}
			error={(submitted || labelTouched) && !photo.label.trim()
				? 'Enter a photo label.'
				: undefined}
		/>
		<TextField
			name="photo-alt"
			label="Alt text"
			placeholder="e.g. White SUV viewed from the front left"
			value={photo.alt_text}
			oninput={(event) => onedit('alt_text', event.currentTarget.value)}
			maxlength={200}
			required
			{disabled}
			onblur={() => (altTouched = true)}
			error={(submitted || altTouched) && !photo.alt_text.trim()
				? 'Describe what the photo shows.'
				: undefined}
		/>
	</div>
</div>
