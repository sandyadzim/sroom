<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import CarSelectField from './CarSelectField.svelte';
	import { carDetailQuery } from '#lib/queries/car.ts';
	import { carVariantName } from '#lib/utils/car.ts';

	let {
		name,
		label,
		value = '',
		placeholder,
		onchange
	}: {
		name: string;
		label: string;
		value?: string;
		placeholder?: string;
		onchange: (value: string) => void;
	} = $props();

	const selected = createQuery(() => ({
		...carDetailQuery(value),
		enabled: !!value
	}));
	const options = $derived(
		selected.data
			? [
					{
						value,
						label:
							carVariantName(selected.data) +
							(selected.data.status === 'inactive' ? ' · Inactive' : '')
					}
				]
			: []
	);
</script>

<CarSelectField {name} {label} {value} {options} {placeholder} {onchange} />
