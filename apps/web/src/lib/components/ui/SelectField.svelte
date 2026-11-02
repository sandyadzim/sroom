<script lang="ts">
	import { Select } from 'bits-ui';
	import { Check, ChevronDown, ChevronUp } from '@lucide/svelte';

	type Props = Omit<Select.TriggerProps, 'value' | 'type' | 'children' | 'child' | 'onchange'> & {
		label: string;
		name: string;
		value?: string;
		options: readonly { value: string; label: string; disabled?: boolean }[];
		placeholder?: string;
		required?: boolean;
		help?: string;
		error?: string;
		onchange?: (value: string) => void;
	};

	const generatedId = $props.id();
	let {
		id = generatedId,
		label,
		name,
		value = $bindable(''),
		options,
		placeholder = 'Select an option',
		disabled = false,
		required = false,
		help,
		error,
		onchange,
		class: className,
		'aria-describedby': describedBy,
		...props
	}: Props = $props();
	const selectedLabel = $derived(options.find((option) => option.value === value)?.label);
	const description = $derived(
		[describedBy, help && `${id}-help`, error && `${id}-error`].filter(Boolean).join(' ') ||
			undefined
	);

	function selectValue(next: string) {
		// Publish the binding before callers prepare a filter request.
		value = next;
		onchange?.(next);
	}
</script>

<div class="min-w-0">
	<label id={`${id}-label`} for={id} class="mb-2 block text-sm font-medium">
		{label}{#if required}<span class="ml-1 text-negative" aria-hidden="true">*</span>{/if}
	</label>
	<Select.Root
		type="single"
		{name}
		disabled={disabled ?? false}
		{required}
		items={[...options]}
		bind:value
		onValueChange={selectValue}
	>
		<Select.Trigger
			{...props}
			{id}
			aria-labelledby={`${id}-label ${id}-value`}
			aria-describedby={description}
			aria-invalid={error ? 'true' : undefined}
			class={[
				'group flex min-h-13 w-full items-center justify-between gap-3 rounded-xl border bg-panel px-4 py-3 text-left text-base text-ink transition-colors focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:bg-panel-muted disabled:text-ink-muted motion-reduce:transition-none',
				error
					? 'border-negative hover:border-negative data-[state=open]:border-negative'
					: 'border-line hover:border-ink-muted data-[state=open]:border-ink-muted',
				className
			]}
		>
			<span id={`${id}-value`} class={['min-w-0 truncate', !selectedLabel && 'text-ink-muted']}
				>{selectedLabel ?? placeholder}</span
			>
			<ChevronDown
				size={18}
				strokeWidth={1.5}
				aria-hidden="true"
				class="shrink-0 transition-transform duration-150 ease-out group-data-[state=open]:rotate-180 motion-reduce:transition-none"
			/>
		</Select.Trigger>
		<Select.Portal>
			<Select.Content
				aria-labelledby={`${id}-label`}
				sideOffset={8}
				align="start"
				collisionPadding={{ top: 108, right: 16, bottom: 16, left: 16 }}
				class="select-popover z-50 flex max-h-(--bits-select-content-available-height) w-(--bits-select-anchor-width) flex-col overflow-hidden rounded-xl border border-line bg-panel p-1.5 text-ink shadow-[0_8px_32px_rgba(16,24,32,0.08)] outline-none"
			>
				<Select.ScrollUpButton class="flex h-6 shrink-0 items-center justify-center text-ink-muted"
					><ChevronUp size={16} aria-hidden="true" /></Select.ScrollUpButton
				>
				<Select.Viewport class="max-h-80 min-h-0 overflow-y-auto overscroll-contain">
					{#each options as option (option.value)}
						<Select.Item
							value={option.value}
							label={option.label}
							disabled={option.disabled}
							class="flex min-h-11 cursor-pointer items-center justify-between gap-3 rounded-lg px-3 py-2.5 text-sm/6 outline-none select-none data-disabled:pointer-events-none data-disabled:text-ink-muted data-highlighted:bg-panel-muted data-selected:font-medium"
						>
							{#snippet children({ selected })}
								<span class="min-w-0 wrap-break-word">{option.label}</span>
								<span class="flex size-4 shrink-0 items-center justify-center"
									>{#if selected}<Check
											size={16}
											strokeWidth={1.75}
											aria-hidden="true"
										/>{/if}</span
								>
							{/snippet}
						</Select.Item>
					{/each}
				</Select.Viewport>
				<Select.ScrollDownButton
					class="flex h-6 shrink-0 items-center justify-center text-ink-muted"
					><ChevronDown size={16} aria-hidden="true" /></Select.ScrollDownButton
				>
			</Select.Content>
		</Select.Portal>
	</Select.Root>
	{#if help}<p id={`${id}-help`} class="mt-2 text-sm text-ink-muted">{help}</p>{/if}
	<div aria-live="polite" aria-atomic="true">
		{#if error}<p id={`${id}-error`} class="mt-2 text-sm text-negative">{error}</p>{/if}
	</div>
</div>
