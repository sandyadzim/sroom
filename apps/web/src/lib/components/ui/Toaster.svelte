<script lang="ts">
	import { MediaQuery } from 'svelte/reactivity';
	import { fly } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
	import type { ToastItem } from '#lib/ui/toast.ts';
	import Toast from './Toast.svelte';

	let { toasts, onDismiss }: { toasts: ToastItem[]; onDismiss: (id: number) => void } = $props();
	const reducedMotion = new MediaQuery('(prefers-reduced-motion: reduce)');
	const gap = 8;
	const peek = 8;
	let heights = $state<Record<number, number>>({});
	let hovered = $state(false);
	let focused = $state(false);
	const expanded = $derived(hovered || focused);
	const front = $derived(toasts[0]);

	const stack = $derived.by(() => {
		const frontHeight = front ? (heights[front.id] ?? 56) : 0;
		let offset = 0;
		const cards = toasts.map((item, index) => {
			const height = heights[item.id] ?? 56;
			const bottom = expanded ? offset : frontHeight - height + index * peek;
			offset += height + gap;
			return { item, bottom, scale: expanded ? 1 : 1 - index * 0.045 };
		});
		return {
			cards,
			height: expanded
				? Math.max(0, offset - gap)
				: frontHeight + Math.max(0, cards.length - 1) * peek
		};
	});

	function measure(id: number, height: number | undefined) {
		if (height === undefined) delete heights[id];
		else heights[id] = height;
	}
</script>

<div role="status" aria-atomic="true" class="sr-only">
	{#if front?.kind === 'success'}
		{#key front.id}{front.message}{/key}
	{/if}
</div>
<div role="alert" aria-atomic="true" class="sr-only">
	{#if front && front.kind !== 'success'}
		{#key front.id}{front.message}{/key}
	{/if}
</div>

<section
	aria-label="Notifications"
	class="fixed inset-x-4 bottom-[max(1rem,env(safe-area-inset-bottom))] z-50 sm:left-auto sm:w-96"
	class:pointer-events-none={!front}
	style:height={`${stack.height}px`}
	onpointerenter={(event) => (hovered = event.pointerType === 'mouse')}
	onpointerleave={() => (hovered = false)}
	onfocusin={() => (focused = true)}
	onfocusout={(event) => {
		focused = event.currentTarget.contains(event.relatedTarget as Node | null);
	}}
>
	{#each stack.cards as { item, bottom, scale }, index (item.id)}
		<div
			class="toast-slot pointer-events-none absolute inset-x-0 bottom-0 origin-top"
			style:transform={`translateY(${-bottom}px) scale(${scale})`}
			style:z-index={3 - index}
		>
			<div
				class:pointer-events-auto={index === 0 || expanded}
				bind:offsetHeight={() => heights[item.id], (height) => measure(item.id, height)}
				in:fly={{ y: 20, duration: reducedMotion.current ? 0 : 220, easing: quintOut }}
				out:fly={{ y: 14, duration: reducedMotion.current ? 0 : 160, easing: quintOut }}
			>
				<Toast feedback={item} paused={expanded} onDismiss={() => onDismiss(item.id)} />
			</div>
		</div>
	{/each}
</section>

<style>
	.toast-slot {
		transition: transform 220ms cubic-bezier(0.23, 1, 0.32, 1);
	}

	@media (prefers-reduced-motion: reduce) {
		.toast-slot {
			transition: none;
		}
	}
</style>
