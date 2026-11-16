<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft, Eye, Pencil } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { isAxiosError } from 'axios';
	import { getApiError } from '#lib/api/staff.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import ImagePreviewDialog from '#lib/components/ui/ImagePreviewDialog.svelte';
	import ImageWithPlaceholder from '#lib/components/ui/ImageWithPlaceholder.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import { carDetailQuery } from '#lib/queries/car.ts';
	import type { AdminCarImage } from '#lib/types/catalog.ts';
	import {
		carBodyTypes,
		carCategories,
		carFuelTypes,
		carOptionLabel,
		carTransmissions
	} from '#lib/utils/car.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import CarForm from './CarForm.svelte';

	let { id, editing = false }: { id: string; editing?: boolean } = $props();
	const query = createQuery(() => carDetailQuery(id));
	const car = $derived(query.data);
	const name = $derived(car ? `${car.brand_name} ${car.model}` : 'Car details');
	const title = $derived(editing ? 'Edit car' : name);
	const cover = $derived(car?.images.find((image) => image.kind === 'cover'));
	const gallery = $derived(car?.images.filter((image) => image.kind === 'gallery') ?? []);
	const errorStatus = $derived(
		isAxiosError(query.error) ? query.error.response?.status : undefined
	);
	const specs = $derived(
		car
			? [
					{ label: 'Brand', value: car.brand_name },
					{ label: 'Model', value: car.model },
					{ label: 'Trim', value: car.trim || '—' },
					{ label: 'Category', value: carOptionLabel(carCategories, car.category) },
					{ label: 'Body type', value: carOptionLabel(carBodyTypes, car.body_type) },
					{ label: 'Seats', value: car.seats },
					{ label: 'Transmission', value: carOptionLabel(carTransmissions, car.transmission) },
					{ label: 'Fuel type', value: carOptionLabel(carFuelTypes, car.fuel_type) }
				]
			: []
	);
</script>

<svelte:head><title>{title} — SROOM</title></svelte:head>

<Link
	href={editing && car
		? resolve('/admin/(staff)/cars/[id]', { id })
		: resolve('/admin/(staff)/cars')}
	class="text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" />
	{editing && car ? 'Back to car' : 'Back to cars'}
</Link>
<header class="mt-2 flex flex-wrap items-start justify-between gap-3">
	<div class="min-w-0">
		<h1
			class={[
				'font-bold tracking-tight wrap-break-word',
				editing ? 'text-3xl/10 sm:text-4xl/12' : 'text-2xl/8 sm:text-3xl/10'
			]}
		>
			{title}
		</h1>
		{#if car && (car.trim || editing)}
			<p class="mt-1 text-sm wrap-break-word text-ink-muted">
				{editing ? `${name} ${car.trim}`.trimEnd() : car.trim}
			</p>
		{/if}
	</div>
	{#if car && !editing}
		<div class="flex flex-wrap gap-2">
			<Button
				href={`${resolve('/admin/(staff)/units')}?car=${encodeURIComponent(car.id)}`}
				variant="secondary"
			>
				Manage units
			</Button>
			<Button href={resolve('/admin/(staff)/cars/[id]/edit', { id: car.id })}>
				<Pencil size={16} strokeWidth={1.5} aria-hidden="true" /> Edit car
			</Button>
		</div>
	{/if}
</header>

{#if car && editing}
	{#key car.id}<CarForm {car} />{/key}
{:else if car}
	<div class="@container mt-6 max-w-6xl">
		<div class="grid items-start gap-5 @min-[52rem]:grid-cols-[minmax(0,1.15fr)_minmax(0,1fr)]">
			<section
				aria-labelledby="vehicle-photos"
				class="min-w-0 rounded-card border border-line bg-panel p-4 sm:p-6"
			>
				<h2 id="vehicle-photos" class="mb-4 text-lg font-semibold">Photos</h2>
				{#if cover}
					{@render photo(cover, 'aspect-[16/10]')}
					<p class="mt-2 text-sm wrap-break-word text-ink-muted">
						<span class="font-medium text-ink">Cover</span> · {cover.label}
					</p>
				{:else}
					<ImageWithPlaceholder alt="" class="aspect-16/10 rounded-xl">
						{#snippet fallback()}<p class="text-sm text-ink-muted">No cover photo</p>{/snippet}
					</ImageWithPlaceholder>
				{/if}
				{#if gallery.length}
					<ul aria-label="Gallery photos" class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
						{#each gallery as image (image.id)}
							<li class="min-w-0">
								{@render photo(image, 'aspect-[4/3]')}
								<p class="mt-1 text-xs/5 wrap-break-word text-ink-muted">{image.label}</p>
							</li>
						{/each}
					</ul>
				{:else}
					<p class="mt-4 text-sm text-ink-muted">No gallery photos.</p>
				{/if}
			</section>
			<div class="min-w-0 space-y-5">
				<section
					aria-labelledby="vehicle-details"
					class="rounded-card border border-line bg-panel p-4 sm:p-6"
				>
					<div class="flex flex-wrap items-start justify-between gap-3">
						<div>
							<h2 id="vehicle-details" class="text-lg font-semibold">Vehicle details</h2>
							<p class="mt-2 text-lg font-semibold tabular-nums">
								{formatIDR(car.daily_rate_idr)}
								<span class="text-sm font-normal text-ink-muted">/ day</span>
							</p>
						</div>
						<StatusBadge
							label={car.status === 'active' ? 'Active' : 'Inactive'}
							tone={car.status === 'active' ? 'success' : 'danger'}
						/>
					</div>
					<dl class="mt-4 grid grid-cols-2 gap-x-4 gap-y-3 border-t border-line pt-4 text-sm">
						{#each specs as spec (spec.label)}
							<div class="min-w-0">
								<dt class="text-xs text-ink-muted">{spec.label}</dt>
								<dd class="mt-1 font-medium wrap-break-word">{spec.value}</dd>
							</div>
						{/each}
					</dl>
					<h3 class="mt-5 border-t border-line pt-4 text-sm font-semibold">Description</h3>
					<p class="mt-2 text-sm/6 wrap-break-word whitespace-pre-line text-ink-muted">
						{car.description}
					</p>
				</section>
				<section
					aria-labelledby="vehicle-record"
					class="rounded-card border border-line bg-panel p-4 sm:p-6"
				>
					<h2 id="vehicle-record" class="mb-4 text-lg font-semibold">Catalog record</h2>
					<dl class="space-y-3 text-sm">
						<div>
							<dt class="text-xs text-ink-muted">Slug</dt>
							<dd class="mt-1 wrap-break-word select-all">{car.slug}</dd>
						</div>
						<div>
							<dt class="text-xs text-ink-muted">Car ID</dt>
							<dd class="mt-1 wrap-break-word select-all">{car.id}</dd>
						</div>
						<div>
							<dt class="text-xs text-ink-muted">Created</dt>
							<dd class="mt-1">
								<time datetime={car.created_at}>{formatWitaDateTime(car.created_at)}</time>
							</dd>
						</div>
						<div>
							<dt class="text-xs text-ink-muted">Last updated</dt>
							<dd class="mt-1">
								<time datetime={car.updated_at}>{formatWitaDateTime(car.updated_at)}</time>
							</dd>
						</div>
					</dl>
				</section>
			</div>
		</div>
	</div>
{:else if query.isError}
	<p class="mt-5 text-negative" role="alert">
		{errorStatus === 404 || errorStatus === 400
			? 'Car not found.'
			: (getApiError(query.error)?.message ?? 'Could not load this car.')}
	</p>
	{#if !errorStatus || errorStatus >= 500}
		<Button
			class="mt-3"
			variant="secondary"
			disabled={query.isFetching}
			onclick={() => query.refetch()}>Try again</Button
		>
	{/if}
{:else}
	<p role="status" class="mt-5 text-sm text-ink-muted">Loading car details…</p>
{/if}

{#snippet photo(image: AdminCarImage, aspect: string)}
	{#if image.url}
		<ImagePreviewDialog title={image.label} filename={image.alt_text} src={image.url}>
			{#snippet trigger({ props })}
				<button
					{...props}
					type="button"
					aria-label={`View ${image.label}`}
					class="group/photo relative block w-full rounded-xl"
				>
					<ImageWithPlaceholder
						src={image.url}
						alt={image.alt_text}
						class={[aspect, 'rounded-xl'].join(' ')}
					/>
					<span
						class="absolute right-2 bottom-2 grid size-9 place-items-center rounded-lg bg-panel text-ink pointer-fine:opacity-0 pointer-fine:group-hover/photo:opacity-100 pointer-fine:group-focus-visible/photo:opacity-100"
					>
						<Eye size={16} aria-hidden="true" />
					</span>
				</button>
			{/snippet}
		</ImagePreviewDialog>
	{:else}
		<ImageWithPlaceholder alt={image.alt_text} class={[aspect, 'rounded-xl'].join(' ')} />
	{/if}
{/snippet}
