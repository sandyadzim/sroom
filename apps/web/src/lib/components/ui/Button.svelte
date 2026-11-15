<script lang="ts">
	import type { HTMLAnchorAttributes, HTMLButtonAttributes } from 'svelte/elements';
	import Link from './Link.svelte';

	type Props = (
		| (HTMLButtonAttributes & { href?: never })
		| (HTMLAnchorAttributes & { href: string; disabled?: never })
	) & {
		variant?: 'primary' | 'secondary' | 'ghost' | 'inverse' | 'ghost-inverse' | 'danger';
		size?: 'default' | 'icon' | 'custom';
	};

	let {
		href,
		children,
		variant = 'primary',
		size = 'default',
		class: className,
		...props
	}: Props = $props();

	const variants = {
		primary: 'bg-primary text-on-primary hover:bg-ink',
		secondary: 'bg-panel-muted text-ink hover:bg-line',
		ghost: 'bg-transparent text-ink hover:bg-panel-muted',
		inverse: 'bg-panel text-primary hover:bg-panel-muted',
		'ghost-inverse': 'bg-transparent text-on-primary hover:bg-on-primary/10',
		danger: 'border border-negative/25 bg-negative/5 text-negative hover:bg-negative/10'
	};
	const sizes = {
		default: 'rounded-full px-5 py-3',
		icon: 'size-11 shrink-0 rounded-lg p-2',
		custom: ''
	};
	const classes = $derived([
		'inline-flex min-h-11 items-center justify-center gap-3 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:bg-panel-muted disabled:text-ink-muted motion-reduce:transition-none',
		variants[variant],
		sizes[size],
		className
	]);
</script>

{#if href !== undefined}
	<Link {href} {...props as HTMLAnchorAttributes} variant="plain" class={classes}>
		{@render children?.()}
	</Link>
{:else}
	<button
		{...props as HTMLButtonAttributes}
		type={(props as HTMLButtonAttributes).type ?? 'button'}
		class={classes}
	>
		{@render children?.()}
	</button>
{/if}
