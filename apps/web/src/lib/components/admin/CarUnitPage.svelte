<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft, Pencil } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { isAxiosError } from 'axios';
	import { getApiError } from '#lib/api/staff.ts';
	import AdminDetailRow from '#lib/components/admin/AdminDetailRow.svelte';
	import AdminDetailSection from '#lib/components/admin/AdminDetailSection.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import { carUnitQuery } from '#lib/queries/car-unit.ts';
	import { carDetailQuery } from '#lib/queries/car.ts';
	import { carVariantName } from '#lib/utils/car.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import CarUnitForm from './CarUnitForm.svelte';

	let { id, editing = false }: { id: string; editing?: boolean } = $props();
	const query = createQuery(() => carUnitQuery(id));
	const unit = $derived(query.data);
	const carId = $derived(unit?.car_id ?? '');
	const parent = createQuery(() => ({ ...carDetailQuery(carId), enabled: Boolean(carId) }));
	const carName = $derived(parent.data ? carVariantName(parent.data) : undefined);
	const title = $derived(editing ? 'Edit unit' : (unit?.internal_code ?? 'Unit details'));
	const listHref = $derived(
		`${resolve('/admin/(staff)/units')}${carId ? `?car=${encodeURIComponent(carId)}` : ''}`
	);
	const errorStatus = $derived(
		isAxiosError(query.error) ? query.error.response?.status : undefined
	);
</script>

<svelte:head><title>{title} — SROOM</title></svelte:head>

<Link
	href={editing && unit ? resolve('/admin/(staff)/units/[id]', { id }) : listHref}
	class="text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" />
	{editing && unit ? 'Back to unit' : 'Back to physical units'}
</Link>
<header class="mt-2 flex flex-wrap items-center justify-between gap-3">
	<h1 class="min-w-0 text-3xl/10 font-bold tracking-tight wrap-anywhere sm:text-4xl/12">{title}</h1>
	{#if unit && !editing}
		<Button href={resolve('/admin/(staff)/units/[id]/edit', { id })}>
			<Pencil size={16} strokeWidth={1.5} aria-hidden="true" /> Edit unit
		</Button>
	{/if}
</header>

{#if unit && editing}
	{#key unit.id}<CarUnitForm {unit} carLabel={carName} />{/key}
{:else if unit}
	<AdminDetailSection title="Unit details">
		<AdminDetailRow label="Car variant">
			<Link
				href={resolve('/admin/(staff)/cars/[id]', { id: unit.car_id })}
				variant="plain"
				class="text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
			>
				{carName ?? (parent.isPending ? 'Loading variant…' : unit.car_id)}
			</Link>
		</AdminDetailRow>
		<AdminDetailRow label="Status">
			<StatusBadge
				label={unit.status === 'active' ? 'Active' : 'Inactive'}
				tone={unit.status === 'active' ? 'success' : 'danger'}
			/>
		</AdminDetailRow>
		<AdminDetailRow label="Unit ID"><span class="select-all">{unit.id}</span></AdminDetailRow>
		<AdminDetailRow label="Created at">
			<time datetime={unit.created_at} class="tabular-nums"
				>{formatWitaDateTime(unit.created_at)}</time
			>
		</AdminDetailRow>
		<AdminDetailRow label="Last updated">
			<time datetime={unit.updated_at} class="tabular-nums"
				>{formatWitaDateTime(unit.updated_at)}</time
			>
		</AdminDetailRow>
	</AdminDetailSection>
{:else if query.isError}
	<div class="mt-6 border-t border-line pt-6">
		<p class="text-negative" role="alert">
			{errorStatus === 400 || errorStatus === 404
				? 'Unit not found.'
				: (getApiError(query.error)?.message ?? 'Could not load this unit.')}
		</p>
		{#if !errorStatus || errorStatus >= 500}
			<Button
				class="mt-3"
				variant="secondary"
				disabled={query.isFetching}
				onclick={() => query.refetch()}
			>
				{query.isFetching ? 'Loading…' : 'Try again'}
			</Button>
		{/if}
	</div>
{:else}
	<p class="mt-6 border-t border-line pt-6 text-ink-muted" role="status">Loading unit details…</p>
{/if}
