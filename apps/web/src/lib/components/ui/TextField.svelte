<script lang="ts">
	import { Eye, EyeOff, X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import type { HTMLInputAttributes } from 'svelte/elements';
	import Button from './Button.svelte';

	type Props = Omit<HTMLInputAttributes, 'value' | 'type'> & {
		label: string;
		value?: string;
		type?: 'text' | 'email' | 'password' | 'tel' | 'url' | 'search' | 'date' | 'time';
		help?: string;
		error?: string;
		revealPassword?: boolean;
		leading?: Snippet;
		prepend?: Snippet;
		append?: Snippet;
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
		prepend,
		append,
		onclear,
		class: className,
		'aria-describedby': describedBy,
		...props
	}: Props = $props();
	let input: HTMLInputElement;
	let passwordVisible = $state(false);
	const canReveal = $derived(type === 'password' && revealPassword);
	const hasAffix = $derived(!!(prepend || append));
	const description = $derived(
		[
			describedBy,
			prepend && `${id}-prepend`,
			append && `${id}-append`,
			help && `${id}-help`,
			error && `${id}-error`
		]
			.filter(Boolean)
			.join(' ') || undefined
	);
</script>

<div class="min-w-0">
	<label for={id} class="mb-2 block text-sm font-medium">
		{label}{#if required}<span class="ml-1 text-negative" aria-hidden="true">*</span>{/if}
	</label>
	<div
		class={hasAffix
			? [
					'flex items-center rounded-xl border transition-colors has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-focus-ring motion-reduce:transition-none',
					error ? 'border-negative' : 'border-line hover:border-ink-muted',
					props.disabled ? 'bg-panel-muted' : 'bg-panel'
				]
			: undefined}
	>
		{#if prepend}
			<span id={`${id}-prepend`} class="shrink-0 pl-4 text-sm text-ink-muted"
				>{@render prepend()}</span
			>
		{/if}
		<div class={['relative min-w-0', hasAffix && 'flex-1']}>
			{#if leading}
				<span
					class="pointer-events-none absolute inset-y-0 left-4 flex items-center text-ink-muted"
				>
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
					'w-full px-4 py-3 text-base text-ink placeholder:text-ink-muted disabled:cursor-not-allowed disabled:text-ink-muted',
					hasAffix
						? 'min-h-[calc(var(--spacing)*13-2px)] bg-transparent focus-visible:outline-none'
						: [
								'min-h-13 rounded-xl border bg-panel transition-colors focus-visible:outline-offset-2 disabled:bg-panel-muted motion-reduce:transition-none',
								error ? 'border-negative' : 'border-line hover:border-ink-muted'
							],
					prepend && 'pl-2',
					append && 'pr-2',
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
		{#if append}
			<span id={`${id}-append`} class="shrink-0 pr-4 text-sm text-ink-muted"
				>{@render append()}</span
			>
		{/if}
	</div>
	{#if help}<p id={`${id}-help`} class="mt-2 text-sm text-ink-muted">{help}</p>{/if}
	<div aria-live="polite" aria-atomic="true">
		{#if error}<p id={`${id}-error`} class="mt-2 text-sm text-negative">{error}</p>{/if}
	</div>
</div>
