<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { tick, untrack } from 'svelte';
	import { carUnitsEndpoint, createCarUnit, updateCarUnit } from '#lib/api/catalog/car-unit.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import AdminFormActions from '#lib/components/admin/AdminFormActions.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { carUnitQuery } from '#lib/queries/car-unit.ts';
	import { carOptionsQuery } from '#lib/queries/car.ts';
	import type { CarUnit } from '#lib/types/catalog.ts';
	import { useToast } from '#lib/ui/toast.ts';
	import { changedUnitFields, unitCodeError } from '#lib/utils/car-unit.ts';
	import { carStatuses, carVariantName } from '#lib/utils/car.ts';
	import CarSelectField from './CarSelectField.svelte';

	let {
		unit,
		carLabel,
		initialCarId = ''
	}: { unit?: CarUnit; carLabel?: string; initialCarId?: string } = $props();
	const toast = useToast();
	const queryClient = useQueryClient();
	const formId = $props.id();
	const cars = createQuery(() => ({ ...carOptionsQuery, enabled: !unit }));
	const carOptions = $derived(
		cars.data?.map((car) => ({
			value: car.id,
			label: carVariantName(car) + (car.status === 'inactive' ? ' · Inactive' : '')
		})) ?? []
	);
	let carId = $state(untrack(() => unit?.car_id ?? initialCarId));
	let code = $state(untrack(() => unit?.internal_code ?? ''));
	let status = $state<CarUnit['status']>(untrack(() => unit?.status ?? 'active'));
	let carError = $state<string>();
	let codeError = $state<string>();
	const input = $derived({ internal_code: code.trim(), status });
	const changes = $derived(unit ? changedUnitFields(unit, input) : {});
	const save = createMutation(() => ({
		mutationFn: () => (unit ? updateCarUnit(unit.id, changes) : createCarUnit(carId, input))
	}));
	const cancelHref = $derived(
		unit
			? resolve('/admin/(staff)/units/[id]', { id: unit.id })
			: `${resolve('/admin/(staff)/units')}${initialCarId ? `?car=${encodeURIComponent(initialCarId)}` : ''}`
	);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (save.isPending) return;
		carError =
			unit || carOptions.some((option) => option.value === carId)
				? undefined
				: 'Select a car variant.';
		codeError = unitCodeError(code);
		if (carError || codeError) {
			await tick();
			document.getElementById(`${formId}-${carError ? 'car' : 'code'}`)?.focus();
			return;
		}
		if (unit && !Object.keys(changes).length) return;

		try {
			const { data, message } = await save.mutateAsync();
			queryClient.setQueryData(carUnitQuery(data.id).queryKey, data);
			void queryClient.invalidateQueries({ queryKey: [carUnitsEndpoint(data.car_id)] });
			toast({ kind: 'success', message });
			await goto(resolve('/admin/(staff)/units/[id]', { id: data.id }));
		} catch (cause) {
			const error = getApiError(cause);
			if (error?.code === 'CAR_UNIT_CODE_TAKEN') codeError = error.message;
			if (error?.code === 'CAR_NOT_FOUND' && !unit) {
				carError = error.message;
				void cars.refetch();
			}
			toast({
				kind: 'error',
				message: error?.message ?? 'Could not save the unit. Please try again.'
			});
			if (codeError || carError) {
				await tick();
				document.getElementById(`${formId}-${carError ? 'car' : 'code'}`)?.focus();
			}
		}
	}
</script>

<form
	method="post"
	novalidate
	onsubmit={submit}
	aria-busy={save.isPending}
	class="mt-6 max-w-lg border-t border-line pt-6"
>
	<div class="space-y-5">
		{#if unit}
			<div>
				<p class="text-sm font-medium">Car variant</p>
				<p class="mt-1 text-sm wrap-anywhere text-ink-muted">{carLabel ?? unit.car_id}</p>
			</div>
		{:else}
			<CarSelectField
				id={`${formId}-car`}
				name="car_id"
				label="Car variant"
				placeholder={cars.isPending ? 'Loading car variants…' : 'Select a car variant'}
				options={carOptions}
				bind:value={carId}
				error={carError}
				disabled={save.isPending || !carOptions.length}
				onchange={() => (carError = undefined)}
				required
			/>
			{#if cars.isError && !carOptions.length}
				<p class="text-sm text-negative" role="alert">
					{getApiError(cars.error)?.message ?? 'Could not load car variants.'}
				</p>
				<Button variant="secondary" disabled={cars.isFetching} onclick={() => cars.refetch()}
					>Try again</Button
				>
			{:else if cars.isSuccess && !carOptions.length}
				<p class="text-sm text-ink-muted">Create a car variant before adding a physical unit.</p>
				<Button variant="secondary" href={resolve('/admin/(staff)/cars/new')}>Create car</Button>
			{/if}
		{/if}

		<TextField
			id={`${formId}-code`}
			name="internal_code"
			label="Internal code"
			placeholder="e.g. INNOVA-001"
			help="Use a unique code for this physical unit."
			value={code}
			error={codeError}
			disabled={save.isPending}
			autocomplete="off"
			oninput={(event) => {
				code = event.currentTarget.value;
				if (codeError) codeError = unitCodeError(code);
			}}
			onblur={() => (codeError ??= unitCodeError(code))}
			required
		/>
		<SelectField
			name="status"
			label="Status"
			options={carStatuses}
			value={status}
			onchange={(value) => (status = value === 'active' ? 'active' : 'inactive')}
			disabled={save.isPending}
			required
		/>
	</div>
	<AdminFormActions>
		<Button href={cancelHref} variant="secondary">Cancel</Button>
		<Button
			type="submit"
			disabled={save.isPending || (unit ? !Object.keys(changes).length : !carOptions.length)}
		>
			{save.isPending ? 'Saving…' : unit ? 'Save changes' : 'Create unit'}
		</Button>
	</AdminFormActions>
</form>
