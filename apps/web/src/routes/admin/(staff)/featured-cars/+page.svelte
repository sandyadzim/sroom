<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowDown, ArrowUp, Eye, Plus, Trash2 } from '@lucide/svelte';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { tick } from 'svelte';
	import { updateFeaturedCars } from '#lib/api/catalog/featured-car.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Dialog from '#lib/components/ui/Dialog.svelte';
	import ImageWithPlaceholder from '#lib/components/ui/ImageWithPlaceholder.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { activeCarOptionsQuery } from '#lib/queries/car.ts';
	import { featuredCarsQuery } from '#lib/queries/featured-car.ts';
	import type { AdminFeaturedCar } from '#lib/types/catalog.ts';
	import { useToast } from '#lib/ui/toast.ts';
	import { formatIDR } from '#lib/utils/currency.ts';

	const ranks = ['1st', '2nd', '3rd'];
	const maxCars = ranks.length;
	const directions = [
		{ step: 1, label: 'down', icon: ArrowDown },
		{ step: -1, label: 'up', icon: ArrowUp }
	] as const;
	const queryClient = useQueryClient();
	const toast = useToast();
	const featured = createQuery(() => featuredCarsQuery);
	const save = createMutation(() => ({
		mutationFn: updateFeaturedCars,
		onSuccess: async ({ data, message }) => {
			await queryClient.cancelQueries({ queryKey: featuredCarsQuery.queryKey });
			queryClient.setQueryData(featuredCarsQuery.queryKey, data);
			draft = null;
			toast({ kind: 'success', message });
			await tick();
			selectionHeading?.focus();
		}
	}));
	const searchId = $props.id();
	let selectionHeading = $state<HTMLHeadingElement>();
	let list = $state<HTMLOListElement>();
	let draft = $state.raw<(AdminFeaturedCar | null)[] | null>(null);
	let search = $state('');
	let appliedSearch = $state('');
	let pickerSlot = $state<number | null>(null);
	let announcement = $state('');
	const saved = $derived(
		ranks.map((_, index) => featured.data?.find((car) => car.position === index + 1) ?? null)
	);
	const selected = $derived(draft ?? saved);
	const selectionCount = $derived(selected.filter((car) => car !== null).length);
	const dirty = $derived(selected.some((car, index) => car?.id !== saved[index]?.id));
	const cars = createQuery(() => ({
		...activeCarOptionsQuery(appliedSearch),
		enabled: pickerSlot !== null
	}));
	const available = $derived(
		cars.data?.filter((car) => !selected.some((item) => item?.id === car.id)) ?? []
	);
	const waitingForSearch = $derived(search.trim() !== appliedSearch || cars.isFetching);

	$effect(() => {
		const keyword = search.trim();
		const timer = setTimeout(() => (appliedSearch = keyword), keyword ? 300 : 0);
		return () => clearTimeout(timer);
	});

	function carName(car: AdminFeaturedCar) {
		return [car.brand_name, car.model, car.trim].filter(Boolean).join(' ');
	}

	async function add(car: AdminFeaturedCar, index: number) {
		if (selected[index] || save.isPending || selected.some((item) => item?.id === car.id)) return;
		const next = [...selected];
		next[index] = car;
		draft = next;
		search = '';
		pickerSlot = null;
		announcement = `${carName(car)} added at position ${index + 1}.`;
		await tick();
		list?.children[index]?.querySelector<HTMLAnchorElement>('[data-view]')?.focus();
	}

	async function move(index: number, step: -1 | 1) {
		const destination = index + step;
		const car = selected[index];
		if (!car || destination < 0 || destination >= maxCars || save.isPending) return;
		const next = [...selected];
		next[index] = next[destination];
		next[destination] = car;
		draft = next;
		announcement = `${carName(car)} moved to position ${destination + 1}.`;
		await tick();
		const row = list?.children[destination];
		const direction = destination === 0 ? 1 : destination === maxCars - 1 ? -1 : step;
		row?.querySelector<HTMLButtonElement>(`[data-step="${direction}"]`)?.focus();
	}

	async function remove(index: number) {
		const car = selected[index];
		if (!car || save.isPending) return;
		const next = [...selected];
		next[index] = null;
		draft = next;
		announcement = `${carName(car)} removed.`;
		await tick();
		list?.children[index]?.querySelector<HTMLButtonElement>('[data-add]')?.focus();
	}

	async function discard() {
		draft = null;
		announcement = 'Changes discarded.';
		await tick();
		selectionHeading?.focus();
	}

	async function submit() {
		if (!dirty || save.isPending) return;
		try {
			await save.mutateAsync({ car_ids: selected.map((car) => car?.id ?? null) });
		} catch (cause) {
			toast({
				kind: 'error',
				message: getApiError(cause)?.message ?? 'Could not save Featured Cars. Please try again.'
			});
		}
	}
</script>

<svelte:head>
	<title>Featured Cars — SROOM</title>
</svelte:head>

<div class="max-w-6xl">
	<header class="flex flex-wrap items-end justify-between gap-x-6 gap-y-4">
		<div>
			<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Featured Cars</h1>
			<p class="mt-2 text-sm/6 text-ink-muted">
				Choose up to three cars and set their display order.
			</p>
		</div>
		{#if featured.data}
			<div class="ml-auto flex flex-wrap justify-end gap-2">
				{#if dirty}
					<Button variant="ghost" disabled={save.isPending} onclick={discard}
						>Discard changes</Button
					>
				{/if}
				<Button disabled={!dirty || save.isPending} onclick={submit}>
					{save.isPending ? 'Saving…' : 'Save changes'}
				</Button>
			</div>
		{/if}
	</header>

	{#if featured.data}
		<section
			class="slot-layout mt-6 border-t border-line pt-5"
			aria-labelledby="featured-selection"
		>
			<div class="mb-3 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
				<h2
					id="featured-selection"
					bind:this={selectionHeading}
					tabindex="-1"
					class="text-base font-semibold"
				>
					Display slots
				</h2>
				<p class="text-sm text-ink-muted tabular-nums">{selectionCount} of {maxCars} selected</p>
			</div>

			<ol bind:this={list} role="list" aria-label="Featured display order" class="grid gap-3">
				{#each ranks as rank, index (selected[index]?.id ?? rank)}
					{@const car = selected[index]}
					{@const name = car ? carName(car) : ''}
					<li
						class={[
							'slot-row grid min-h-26 items-center gap-x-3 gap-y-2 rounded-xl border border-line p-3 sm:px-4',
							car ? 'bg-panel' : 'slot-empty border-dashed bg-page'
						]}
					>
						<div class="slot-reorder flex gap-1">
							{#each directions as { step, label, icon: Icon } (step)}
								<Button
									variant="ghost"
									size="icon"
									class="rounded-full border border-line disabled:bg-transparent disabled:text-ink-muted/50"
									data-step={step}
									aria-label={car ? `Move ${name} ${label}` : `Move empty slot ${label}`}
									title={`Move ${label}`}
									disabled={!car || index + step < 0 || index + step >= maxCars || save.isPending}
									onclick={() => move(index, step)}
								>
									<Icon size={17} strokeWidth={1.5} aria-hidden="true" />
								</Button>
							{/each}
						</div>
						<div
							class="slot-rank min-w-10 justify-self-start rounded-md bg-primary px-2 py-1.5 text-center text-xs font-semibold text-on-primary tabular-nums"
						>
							<span aria-hidden="true">{rank}</span>
							<span class="sr-only">Position {index + 1}</span>
						</div>
						{#if car}
							<div class="slot-car flex min-w-0 items-center gap-3">
								<ImageWithPlaceholder
									src={car.cover?.url}
									alt=""
									class="aspect-4/3 w-14 shrink-0 rounded-lg sm:w-20"
								/>
								<div class="min-w-0 wrap-anywhere">
									<p class="font-semibold tracking-tight">{car.brand_name} {car.model}</p>
									{#if car.trim}<p class="mt-0.5 text-sm text-ink-muted">{car.trim}</p>{/if}
								</div>
							</div>
							<div class="slot-meta grid justify-items-start gap-1">
								<p class="text-sm font-semibold tabular-nums">
									{formatIDR(car.daily_rate_idr)}
									<span class="text-xs font-normal text-ink-muted">/ day</span>
								</p>
								<StatusBadge
									label={car.status === 'active' ? 'Active' : 'Inactive'}
									tone={car.status === 'active' ? 'success' : 'danger'}
								/>
								{#if car.status === 'inactive'}
									<span class="sr-only">Excluded from public selection.</span>
								{/if}
							</div>
							<div class="slot-actions flex justify-end gap-1">
								<Button
									href={resolve('/admin/(staff)/cars/[id]', { id: car.id })}
									variant="secondary"
									size="icon"
									class="rounded-full"
									data-view
									title="View car"
									aria-label={`View ${name}`}
									><Eye size={18} strokeWidth={1.5} aria-hidden="true" /></Button
								>
								<Button
									variant="danger"
									size="icon"
									class="rounded-full"
									aria-label={`Remove ${name}`}
									title="Remove car"
									disabled={save.isPending}
									onclick={() => remove(index)}
									><Trash2 size={17} strokeWidth={1.5} aria-hidden="true" /></Button
								>
							</div>
						{:else}
							<p class="slot-car text-sm font-semibold text-ink-muted">Empty featured slot</p>
							<div class="slot-actions flex justify-end">
								<Dialog
									title="Add an active car"
									description={`Choose a car for the ${rank} display slot.`}
									size="compact"
									bind:open={
										() => pickerSlot === index,
										(open) => {
											if (open) pickerSlot = index;
											else if (pickerSlot === index) pickerSlot = null;
										}
									}
								>
									{#snippet trigger({ props })}
										<Button
											{...props}
											variant="secondary"
											size="icon"
											class="rounded-full"
											data-add
											aria-label={`Add car to ${rank} slot`}
											title="Add car"
											disabled={save.isPending}
											><Plus size={18} strokeWidth={1.5} aria-hidden="true" /></Button
										>
									{/snippet}
									<TextField
										id={`${searchId}-${index}`}
										type="search"
										label="Search active cars"
										placeholder="Search by make, model, or trim"
										value={search}
										oninput={(event) => (search = event.currentTarget.value)}
									/>
									{#if waitingForSearch || cars.isPending}
										<p role="status" class="mt-3 text-sm text-ink-muted">Loading active cars…</p>
									{:else if cars.isError}
										<p role="alert" class="mt-3 text-sm text-negative">
											{getApiError(cars.error)?.message ?? 'Could not load active cars.'}
										</p>
										<Button
											class="mt-3"
											variant="secondary"
											disabled={cars.isFetching}
											onclick={() => cars.refetch()}
										>
											Try again
										</Button>
									{:else if available.length}
										<ul
											class="mt-3 max-h-64 divide-y divide-line overflow-y-auto overscroll-contain px-1"
										>
											{#each available as option (option.id)}
												<li>
													<Button
														variant="ghost"
														size="custom"
														class="w-full justify-start rounded-lg px-2 py-2 text-left focus-visible:-outline-offset-2"
														aria-label={`Add ${carName(option)}`}
														onclick={() => add(option, index)}
													>
														<ImageWithPlaceholder
															src={option.cover?.url}
															alt=""
															class="aspect-4/3 w-14 shrink-0 rounded-md"
														/>
														<span class="min-w-0 flex-1 wrap-anywhere">{carName(option)}</span>
														<Plus size={16} class="shrink-0" aria-hidden="true" />
													</Button>
												</li>
											{/each}
										</ul>
									{:else}
										<p role="status" class="mt-3 text-sm text-ink-muted">
											{appliedSearch
												? 'No cars match your search.'
												: 'No more active cars available.'}
										</p>
									{/if}
								</Dialog>
							</div>
						{/if}
					</li>
				{/each}
			</ol>
			<p class="sr-only" role="status">{announcement}</p>
		</section>
	{:else if featured.isError}
		<p class="mt-6 text-sm text-negative" role="alert">
			{getApiError(featured.error)?.message ?? 'Could not load Featured Cars.'}
		</p>
		<Button
			class="mt-3"
			variant="secondary"
			disabled={featured.isFetching}
			onclick={() => featured.refetch()}>Try again</Button
		>
	{:else}
		<p class="mt-6 text-sm text-ink-muted" role="status">Loading Featured Cars…</p>
	{/if}
</div>

<style>
	.slot-layout {
		container-type: inline-size;
	}

	.slot-row {
		grid-template-columns: auto minmax(0, 1fr) auto;
		grid-template-areas: 'rank car car' '. meta meta' 'reorder reorder actions';
	}

	.slot-empty {
		grid-template-areas: 'rank car actions';
	}

	.slot-empty .slot-reorder {
		display: none;
	}

	.slot-reorder {
		grid-area: reorder;
	}
	.slot-rank {
		grid-area: rank;
	}
	.slot-car {
		grid-area: car;
	}
	.slot-meta {
		grid-area: meta;
	}
	.slot-actions {
		grid-area: actions;
	}

	@container (width >= 22rem) {
		.slot-row {
			grid-template-areas: 'rank car car' 'reorder meta actions';
		}
		.slot-empty {
			grid-template-areas: 'rank car actions';
		}
	}

	@container (width >= 42rem) {
		.slot-row {
			grid-template-columns: auto auto minmax(0, 1fr) minmax(9.5rem, auto) auto;
			grid-template-areas: 'reorder rank car meta actions';
		}
		.slot-empty .slot-reorder {
			display: flex;
		}
		.slot-meta {
			border-inline: 1px solid var(--color-line);
			padding-inline: 1rem;
		}
	}
</style>
