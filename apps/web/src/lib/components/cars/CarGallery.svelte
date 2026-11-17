<script lang="ts">
	import { Expand, ChevronLeft, ChevronRight } from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Dialog from '#lib/components/ui/Dialog.svelte';
	import type { CarGalleryImage } from '#lib/types/car.ts';

	let { name, images }: { name: string; images: readonly CarGalleryImage[] } = $props();
	let selected = $state(0);
	let failedImages = $state<string[]>([]);
	const image = $derived(images[selected] ?? images[0]);
	const id = $props.id();

	function changeImage(direction: number) {
		if (images.length < 2) return;
		selected = (selected + direction + images.length) % images.length;
	}

	function navigateDialog(event: KeyboardEvent) {
		if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
		event.preventDefault();
		changeImage(event.key === 'ArrowRight' ? 1 : -1);
	}

	function moveImage(event: KeyboardEvent, index: number) {
		let next: number;
		if (event.key === 'ArrowRight') next = (index + 1) % images.length;
		else if (event.key === 'ArrowLeft') next = (index - 1 + images.length) % images.length;
		else if (event.key === 'Home') next = 0;
		else if (event.key === 'End') next = images.length - 1;
		else return;
		event.preventDefault();
		selected = next;
		document.getElementById(`${id}-${next}`)?.focus();
	}
</script>

{#snippet artwork(enlarged = false)}
	{#if !image || failedImages.includes(image.src)}
		<div
			role="img"
			aria-label={`Image unavailable for ${name}`}
			class="flex aspect-16/10 items-center justify-center rounded-card bg-panel-muted p-6 text-center text-ink-muted"
		>
			Image unavailable. You can still explore the car details below.
		</div>
	{:else}
		<img
			src={image.src}
			alt={image.alt}
			width="1536"
			height="1024"
			fetchpriority={enlarged ? 'auto' : 'high'}
			onerror={() => (failedImages = [...failedImages, image.src])}
			class={[
				'w-full rounded-card bg-panel-muted object-contain',
				enlarged ? 'max-h-[65dvh]' : 'aspect-16/10'
			]}
		/>
	{/if}
{/snippet}

<div aria-label={`${name} images`}>
	<div class="relative overflow-hidden rounded-card bg-panel-muted">
		{@render artwork()}
		{#if image}
			<Dialog
				title={name}
				description="Enlarged vehicle image."
				closeLabel="Close image viewer"
				onkeydown={navigateDialog}
			>
				{#snippet trigger({ props })}
					<Button
						{...props}
						variant="inverse"
						size="icon"
						class="absolute right-3 bottom-3"
						aria-label={`Enlarge ${name} image`}
					>
						<Expand size={18} strokeWidth={1.5} aria-hidden="true" />
					</Button>
				{/snippet}
				{@render artwork(true)}
				<div
					class="mt-4 flex items-center justify-between gap-4"
					role="group"
					aria-label="Gallery navigation"
				>
					<Button
						variant="secondary"
						size="icon"
						aria-label="Previous image"
						disabled={images.length < 2}
						onclick={() => changeImage(-1)}
					>
						<ChevronLeft size={20} strokeWidth={1.5} aria-hidden="true" />
					</Button>
					<p class="text-sm/6 text-ink-muted tabular-nums" aria-live="polite" aria-atomic="true">
						{selected + 1} / {images.length}
					</p>
					<Button
						variant="secondary"
						size="icon"
						aria-label="Next image"
						disabled={images.length < 2}
						onclick={() => changeImage(1)}
					>
						<ChevronRight size={20} strokeWidth={1.5} aria-hidden="true" />
					</Button>
				</div>
			</Dialog>
		{/if}
	</div>
	{#if images.length > 0}
		<div class="mt-3 flex flex-wrap gap-3" role="group" aria-label="Choose a vehicle image">
			{#each images as item, index (`${index}-${item.src}`)}
				<Button
					id={`${id}-${index}`}
					variant="ghost"
					size="custom"
					onclick={() => (selected = index)}
					onkeydown={(event: KeyboardEvent) => moveImage(event, index)}
					aria-label={`Show ${item.label.toLowerCase()} image`}
					aria-pressed={selected === index}
					class={[
						'h-16 w-24 overflow-hidden rounded-lg border-2 bg-panel-muted p-1 transition-colors motion-reduce:transition-none',
						selected === index ? 'border-primary' : 'border-transparent hover:border-ink-muted'
					]}
				>
					{#if failedImages.includes(item.src)}<span class="text-xs">{item.label}</span>{:else}
						<img
							src={item.src}
							alt=""
							width="96"
							height="64"
							loading="lazy"
							onerror={() => (failedImages = [...failedImages, item.src])}
							class="h-full w-full object-contain"
						/>
					{/if}
				</Button>
			{/each}
		</div>
	{/if}
</div>
