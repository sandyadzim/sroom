<script lang="ts">
	import { Check, ChevronDown, Search } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { Combobox } from 'bits-ui';
	import { getApiError } from '#lib/api/staff.ts';
	import { carSearchOptionsQuery } from '#lib/queries/car.ts';
	import { carVariantName } from '#lib/utils/car.ts';

	const generatedId = $props.id();
	let {
		id = generatedId,
		name,
		label = 'Car variant',
		value = $bindable(''),
		options,
		placeholder = 'Select a car variant',
		help,
		error,
		activeOnly = false,
		disabled = false,
		required = false,
		onchange
	}: {
		id?: string;
		name: string;
		label?: string;
		value?: string;
		options: readonly { value: string; label: string }[];
		placeholder?: string;
		help?: string;
		error?: string;
		activeOnly?: boolean;
		disabled?: boolean;
		required?: boolean;
		onchange?: (value: string) => void;
	} = $props();

	let open = $state(false);
	let search = $state('');
	let appliedSearch = $state('');
	const keyword = $derived(search.trim());
	const cars = createQuery(() => ({
		...carSearchOptionsQuery(appliedSearch, activeOnly),
		enabled: open && !!appliedSearch && keyword === appliedSearch,
		refetchOnWindowFocus: false
	}));
	const loading = $derived(!!keyword && (keyword !== appliedSearch || cars.isPending));
	const results = $derived(
		keyword
			? loading || cars.isError
				? []
				: (cars.data?.map((car) => ({
						value: car.id,
						label:
							carVariantName(car) + (!activeOnly && car.status === 'inactive' ? ' · Inactive' : '')
					})) ?? [])
			: options
	);
	const selectedLabel = $derived(options.find((option) => option.value === value)?.label);
	const description = $derived(
		[help && `${id}-help`, error && `${id}-error`].filter(Boolean).join(' ') || undefined
	);

	$effect(() => {
		const term = keyword;
		const timer = setTimeout(() => (appliedSearch = term), term ? 300 : 0);
		return () => clearTimeout(timer);
	});

	function selectCar(next: string) {
		value = next;
		onchange?.(next);
	}
</script>

<div class="min-w-0">
	<label id={`${id}-label`} for={id} class="mb-2 block text-sm font-medium">
		{label}{#if required}<span class="ml-1 text-negative" aria-hidden="true">*</span>{/if}
	</label>
	<Combobox.Root
		type="single"
		{name}
		{required}
		{disabled}
		bind:open
		{value}
		items={[...options]}
		inputValue={open ? search : (selectedLabel ?? '')}
		allowDeselect={false}
		onValueChange={selectCar}
		onOpenChange={(next) => {
			if (!next) search = '';
		}}
	>
		<div class="relative">
			<Combobox.Input
				{id}
				placeholder={open ? (selectedLabel ?? 'Search by make, model, or trim') : placeholder}
				autocomplete="off"
				aria-labelledby={`${id}-label`}
				aria-controls={open ? `${id}-options` : undefined}
				aria-describedby={description}
				aria-invalid={error ? 'true' : undefined}
				aria-required={required}
				class={[
					'min-h-13 w-full rounded-xl border bg-panel px-4 py-3 pr-12 disabled:cursor-not-allowed disabled:bg-panel-muted disabled:text-ink-muted',
					error ? 'border-negative' : 'border-line hover:border-ink-muted'
				]}
				onclick={() => (open = true)}
				oninput={(event) => {
					open = true;
					search = event.currentTarget.value;
				}}
			/>
			<Combobox.Trigger
				tabindex={-1}
				type="button"
				aria-label="Toggle car variants"
				class="absolute inset-y-0 right-0 grid w-12 place-items-center rounded-r-xl text-ink-muted"
			>
				{#if open}<Search size={18} aria-hidden="true" />{:else}<ChevronDown
						size={18}
						aria-hidden="true"
					/>{/if}
			</Combobox.Trigger>
		</div>
		<Combobox.Portal>
			<Combobox.Content
				id={`${id}-options`}
				aria-labelledby={`${id}-label`}
				sideOffset={8}
				align="start"
				collisionPadding={{ top: 108, right: 16, bottom: 16, left: 16 }}
				class="select-popover z-50 flex max-h-(--bits-combobox-content-available-height) w-(--bits-combobox-anchor-width) flex-col overflow-hidden rounded-xl border border-line bg-panel p-1.5 text-ink shadow-[0_8px_32px_rgba(16,24,32,0.08)] outline-none"
			>
				{#if loading}
					<p role="status" class="px-3 py-2 text-sm text-ink-muted">Searching cars…</p>
				{:else if keyword && cars.isError}
					<p role="status" class="px-3 py-2 text-sm text-negative">
						{getApiError(cars.error)?.message ??
							'Could not search cars. Change the search and try again.'}
					</p>
				{:else if results.length}
					<Combobox.Viewport class="max-h-80 min-h-0 overflow-y-auto overscroll-contain">
						{#each results as option (option.value)}
							<Combobox.Item
								value={option.value}
								label={option.label}
								class="flex min-h-11 cursor-pointer items-center justify-between gap-3 rounded-lg px-3 py-2 text-sm/6 outline-none select-none data-highlighted:bg-panel-muted data-selected:font-medium"
							>
								{#snippet children({ selected })}
									<span class="min-w-0 wrap-anywhere">{option.label}</span>
									{#if selected}<Check size={16} class="shrink-0" aria-hidden="true" />{/if}
								{/snippet}
							</Combobox.Item>
						{/each}
					</Combobox.Viewport>
				{:else}
					<p role="status" class="px-3 py-2 text-sm text-ink-muted">No cars match your search.</p>
				{/if}
			</Combobox.Content>
		</Combobox.Portal>
	</Combobox.Root>
	{#if help}<p id={`${id}-help`} class="mt-2 text-sm text-ink-muted">{help}</p>{/if}
	<div aria-live="polite">
		{#if error}<p id={`${id}-error`} class="mt-2 text-sm text-negative">{error}</p>{/if}
	</div>
</div>
