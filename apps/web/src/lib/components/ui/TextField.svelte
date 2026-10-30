<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	type Props = Omit<HTMLInputAttributes, 'value' | 'type'> & {
		label: string;
		name: string;
		value?: string;
		type?: 'text' | 'email' | 'password' | 'tel' | 'url' | 'search';
		help?: string;
		error?: string;
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
		class: className,
		'aria-describedby': describedBy,
		...props
	}: Props = $props();
	const description = $derived(
		[describedBy, help && `${id}-help`, error && `${id}-error`].filter(Boolean).join(' ') ||
			undefined
	);
</script>

<div class="min-w-0">
	<label for={id} class="mb-2 block text-sm font-medium">
		{label}{#if required}<span class="ml-1 text-negative" aria-hidden="true">*</span>{/if}
	</label>
	<input
		{...props}
		{id}
		{type}
		{required}
		bind:value
		aria-invalid={error ? 'true' : undefined}
		aria-describedby={description}
		class={[
			'min-h-13 w-full rounded-xl border bg-panel px-4 py-3 text-base text-ink transition-colors placeholder:text-ink-muted focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:bg-panel-muted disabled:text-ink-muted motion-reduce:transition-none',
			error ? 'border-negative' : 'border-line hover:border-ink-muted',
			className
		]}
	/>
	{#if help}<p id={`${id}-help`} class="mt-2 text-sm text-ink-muted">{help}</p>{/if}
	<div aria-live="polite" aria-atomic="true">
		{#if error}<p id={`${id}-error`} class="mt-2 text-sm text-negative">{error}</p>{/if}
	</div>
</div>
