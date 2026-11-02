<script lang="ts">
	import type { HTMLAnchorAttributes } from 'svelte/elements';
	import Link from '#lib/components/ui/Link.svelte';
	import { resolve } from '$app/paths';

	let {
		size = 'default',
		inverse = false,
		href = resolve('/'),
		class: className,
		...props
	}: HTMLAnchorAttributes & { size?: 'default' | 'display'; inverse?: boolean } = $props();
</script>

<Link
	aria-label="SROOM home"
	{href}
	{...props}
	variant="plain"
	class={[
		size === 'display'
			? '@container block overflow-hidden leading-none focus-visible:-outline-offset-4'
			: [
					'inline-flex min-h-11 items-center font-display text-2xl leading-none font-bold tracking-tight whitespace-nowrap sm:text-3xl',
					inverse ? 'text-on-primary' : 'text-primary'
				],
		className
	]}
>
	<span class={size === 'display' ? 'footer-wordmark' : undefined} aria-hidden="true">SROOM.</span>
</Link>
