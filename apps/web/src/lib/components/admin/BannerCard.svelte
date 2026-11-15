<script lang="ts">
	import { resolve } from '$app/paths';
	import { ImagePlus, Pencil, Plus, Trash2 } from '@lucide/svelte';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { tick } from 'svelte';
	import { clearBannerSlot } from '#lib/api/catalog/banner.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Dialog from '#lib/components/ui/Dialog.svelte';
	import ImageWithPlaceholder from '#lib/components/ui/ImageWithPlaceholder.svelte';
	import { bannerSlotQuery, bannerSlotsQuery } from '#lib/queries/banner.ts';
	import type { BannerSlot } from '#lib/types/catalog.ts';
	import { useToast } from '#lib/ui/toast.ts';

	let { slot }: { slot: BannerSlot } = $props();
	const id = $props.id();
	let confirmOpen = $state(false);
	const queryClient = useQueryClient();
	const toast = useToast();
	const clear = createMutation(() => ({
		mutationFn: clearBannerSlot
	}));

	async function remove() {
		try {
			const { data, message } = await clear.mutateAsync(slot.position);
			confirmOpen = false;
			queryClient.setQueryData(bannerSlotQuery(data.position).queryKey, data);
			queryClient.setQueryData(bannerSlotsQuery.queryKey, (slots) =>
				slots?.map((current) => (current.position === data.position ? data : current))
			);
			void queryClient.invalidateQueries({ queryKey: bannerSlotsQuery.queryKey, exact: true });
			toast({ kind: 'success', message });
			await tick();
			document.getElementById(`${id}-edit`)?.focus();
		} catch (cause) {
			toast({
				kind: 'error',
				message: getApiError(cause)?.message ?? 'Could not remove the banner. Please try again.'
			});
		}
	}
</script>

<article
	class="flex h-full min-w-0 flex-col overflow-hidden rounded-xl border border-line bg-panel"
>
	<div class="flex items-center justify-between gap-2 border-b border-line px-4 py-3 text-sm">
		<h2 class="font-semibold">Hero slot {slot.position}</h2>
		<span class={slot.content ? 'text-primary' : 'text-ink-muted'}>
			{slot.content ? 'Filled' : 'Empty'}
		</span>
	</div>

	{#if slot.content}
		{@const content = slot.content}
		<ImageWithPlaceholder src={content.image_url} alt="" class="aspect-video max-h-56 w-full">
			{#snippet fallback()}
				<span class="text-sm text-ink-muted">Image unavailable</span>
			{/snippet}
		</ImageWithPlaceholder>
		<div class="px-4 pt-4 pb-5 wrap-anywhere">
			<p class="text-sm text-ink-muted">{content.brand_name} {content.car_model}</p>
			<p class="mt-2 text-lg/6 font-semibold">
				{content.headline_line_1}<br />{content.headline_line_2}
			</p>
			<p class="mt-2 text-sm/6 text-ink-muted">{content.description}</p>
		</div>
	{:else}
		<div class="grid aspect-video max-h-56 w-full place-items-center bg-page text-ink-muted">
			<ImagePlus size={28} strokeWidth={1.5} aria-hidden="true" />
		</div>
		<p class="px-4 pt-4 pb-5 text-sm/6 text-ink-muted">
			Add a vehicle, headline, and image to fill this slot.
		</p>
	{/if}

	<div
		class="mt-auto flex flex-wrap items-center justify-between gap-2 border-t border-line px-3 py-2"
	>
		<Button
			id={`${id}-edit`}
			href={resolve('/admin/(staff)/banners/[position]', { position: String(slot.position) })}
			variant={slot.content ? 'ghost' : 'secondary'}
			size="custom"
			class="gap-2 rounded-lg px-3"
			aria-label={`${slot.content ? 'Edit banner' : 'Add banner'}, slot ${slot.position}`}
		>
			{#if slot.content}
				<Pencil size={16} strokeWidth={1.5} aria-hidden="true" /> Edit banner
			{:else}
				<Plus size={16} strokeWidth={1.5} aria-hidden="true" /> Add banner
			{/if}
		</Button>
		{#if slot.content}
			<Dialog
				title={`Remove banner from slot ${slot.position}?`}
				description="Confirm that you want to clear this banner slot."
				size="compact"
				bind:open={confirmOpen}
			>
				{#snippet trigger({ props })}
					<Button
						{...props}
						variant="ghost"
						size="icon"
						class="text-negative"
						disabled={clear.isPending}
						aria-label={`Remove banner, slot ${slot.position}`}
						title="Remove banner"
					>
						<Trash2 size={17} strokeWidth={1.5} aria-hidden="true" />
					</Button>
				{/snippet}
				<p class="text-sm/6 text-ink-muted">
					This clears the content and leaves slot {slot.position} available for another banner.
				</p>
				<div class="mt-5 flex flex-wrap justify-end gap-2">
					<Button
						variant="secondary"
						onclick={() => (confirmOpen = false)}
						disabled={clear.isPending}>Cancel</Button
					>
					<Button variant="danger" onclick={remove} disabled={clear.isPending}>
						{clear.isPending ? 'Removing…' : 'Remove banner'}
					</Button>
				</div>
			</Dialog>
		{/if}
	</div>
</article>
