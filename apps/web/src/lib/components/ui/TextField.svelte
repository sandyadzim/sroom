<script lang="ts">
	import { Eye, EyeOff, X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLInputAttributes } from 'svelte/elements';
	import Button from './Button.svelte';

	type Props = Omit<HTMLInputAttributes, 'value' | 'type'> & {
		label: string;
		name: string;
		value?: string;
		type?: 'text' | 'email' | 'password' | 'tel' | 'url' | 'search';
		help?: string;
		error?: string;
		revealPassword?: boolean;
		leading?: Snippet;
		onclear?: () => void;
	};

	const generatedId = $props.id();
	let {
		id = generatedId,
		label,
		value = $bindable(''),
		type = 'text',
		required = false,
		help,
		error,
		revealPassword = false,
		leading,
		onclear,
		class: className,
		'aria-describedby': describedBy,
		...props
	}: Props = $props();
	let input: HTMLInputElement;
	let passwordVisible = $state(false);
	const canReveal = $derived(type === 'password' && revealPassword);
	const description = $derived(
		[describedBy, help && `${id}-help`, error && `${id}-error`].filter(Boolean).join(' ') ||
			undefined
	);
</script>

<div class="min-w-0">
	<label for={id} class="mb-2 block text-sm font-medium">
		{label}{#if required}<span class="ml-1 text-negative" aria-hidden="true">*</span>{/if}
	</label>
	<div class="relative">
		{#if leading}
			<span class="pointer-events-none absolute inset-y-0 left-4 flex items-center text-ink-muted">
				{@render leading()}
			</span>
		{/if}
		<input
			bind:this={input}
			{...props}
			{id}
			type={canReveal && passwordVisible ? 'text' : type}
			{required}
			bind:value
			aria-invalid={error ? 'true' : undefined}
			aria-describedby={description}
			class={[
				'min-h-13 w-full rounded-xl border bg-panel px-4 py-3 text-base text-ink transition-colors placeholder:text-ink-muted focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:bg-panel-muted disabled:text-ink-muted motion-reduce:transition-none',
				error ? 'border-negative' : 'border-line hover:border-ink-muted',
				leading && 'pl-11',
				(canReveal || onclear) && 'pr-14',
				onclear && '[&::-webkit-search-cancel-button]:appearance-none',
				className
			]}
		/>
		{#if canReveal}
			<Button
				type="button"
				variant="ghost"
				size="icon"
				class="absolute top-1/2 right-1 -translate-y-1/2 text-ink-muted focus-visible:-outline-offset-2"
				aria-label={`${passwordVisible ? 'Hide' : 'Show'} ${label.toLowerCase()}`}
				aria-controls={id}
				disabled={props.disabled}
				onclick={() => (passwordVisible = !passwordVisible)}
			>
				{#if passwordVisible}
					<EyeOff size={18} strokeWidth={1.5} aria-hidden="true" />
				{:else}
					<Eye size={18} strokeWidth={1.5} aria-hidden="true" />
				{/if}
			</Button>
		{:else if onclear && value && !props.disabled}
			<Button
				variant="ghost"
				size="icon"
				class="absolute top-1/2 right-1 -translate-y-1/2 text-ink-muted focus-visible:-outline-offset-2"
				aria-label={`Clear ${label.toLowerCase()}`}
				aria-controls={id}
				onclick={() => {
					value = '';
					onclear?.();
					input.focus();
				}}
			>
				<X size={18} strokeWidth={1.5} aria-hidden="true" />
			</Button>
		{/if}
	</div>
	{#if help}<p id={`${id}-help`} class="mt-2 text-sm text-ink-muted">{help}</p>{/if}
	<div aria-live="polite" aria-atomic="true">
		{#if error}<p id={`${id}-error`} class="mt-2 text-sm text-negative">{error}</p>{/if}
	</div>
</div>
