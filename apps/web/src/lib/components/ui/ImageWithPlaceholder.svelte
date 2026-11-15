<script lang="ts">
	import { Image as ImageIcon, ImageOff } from '@lucide/svelte';
	import type { Snippet } from 'svelte';

	let {
		src,
		alt,
		fit = 'contain',
		class: className,
		fallback
	}: {
		src?: string | null;
		alt: string;
		fit?: 'contain' | 'scale-down';
		class?: string;
		fallback?: Snippet;
	} = $props();

	let image = $state<HTMLImageElement>();
	let status = $state<'loading' | 'ready' | 'error'>('loading');
	const loading = $derived(!!src && status === 'loading');

	$effect(() => {
		const element = image;
		if (!element) return;

		status = 'loading';
		let active = true;
		void element.decode().then(
			() => {
				if (active) status = 'ready';
			},
			() => {
				if (active) status = 'error';
			}
		);
		return () => {
			active = false;
		};
	});
</script>

<div class={['relative grid place-items-center overflow-hidden bg-page', className]}>
	{#if src}
		{#key src}
			<img
				bind:this={image}
				{src}
				{alt}
				aria-hidden={status !== 'ready'}
				class={[
					'absolute inset-0 size-full motion-safe:transition-opacity motion-safe:duration-150',
					fit === 'scale-down' ? 'object-scale-down' : 'object-contain',
					status === 'ready' ? 'opacity-100' : 'opacity-0'
				]}
			/>
		{/key}
	{/if}

	<span role="status" class="sr-only">{loading ? 'Loading image…' : ''}</span>
	{#if loading}
		<div
			class="absolute inset-0 grid place-items-center bg-panel-muted motion-safe:animate-pulse"
			aria-hidden="true"
		>
			<ImageIcon size={24} strokeWidth={1.5} class="text-ink-muted" />
		</div>
	{:else if !src || status === 'error'}
		{#if fallback}
			{@render fallback()}
		{:else}
			<ImageOff size={23} strokeWidth={1.5} class="text-ink-muted" aria-hidden="true" />
		{/if}
	{/if}
</div>
