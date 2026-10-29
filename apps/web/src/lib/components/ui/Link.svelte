<script lang="ts">
	import type { HTMLAnchorAttributes } from 'svelte/elements';
	import { ArrowUpRight } from '@lucide/svelte';

	type Props = HTMLAnchorAttributes & {
		variant?: 'plain' | 'text' | 'nav' | 'footer';
		arrow?: boolean;
	};

	let {
		href = '#',
		children,
		variant = 'text',
		arrow = false,
		class: className,
		...props
	}: Props = $props();

	const variants = {
		plain: '',
		text: 'inline-flex min-h-11 items-center justify-center gap-2 hover:underline',
		nav: 'flex min-h-11 items-center gap-4 whitespace-nowrap',
		footer:
			'inline-flex min-h-11 items-center gap-2 text-xl tracking-tight text-on-primary/85 hover:text-on-primary focus-visible:text-on-primary'
	};
</script>

<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- Native links accept fragments and external URLs; callers resolve application routes. -->
<a {href} {...props} class={['group', variants[variant], className]}>
	{#if variant === 'nav'}
		<span class="nav-label">{@render children?.()}</span>
	{:else}
		{@render children?.()}
	{/if}
	{#if arrow}
		<ArrowUpRight
			size={variant === 'nav' ? 24 : 18}
			strokeWidth={1.5}
			aria-hidden="true"
			class={variant === 'footer'
				? 'shrink-0 opacity-0 transition duration-150 ease-out group-hover:translate-x-1 group-hover:opacity-100 group-focus-visible:opacity-100 motion-reduce:transform-none motion-reduce:transition-none'
				: 'shrink-0'}
		/>
	{/if}
</a>
