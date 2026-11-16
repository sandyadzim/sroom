import { queryOptions } from '@tanstack/svelte-query';
import { carsEndpoint, fetchActiveCarsForSelect, fetchCar } from '#lib/api/catalog/car.ts';

export function activeCarOptionsQuery(search = '') {
	return queryOptions({
		queryKey: [carsEndpoint, 'options', 'active', search],
		queryFn: ({ signal }) => fetchActiveCarsForSelect(search, signal)
	});
}

export function carDetailQuery(id: string) {
	return queryOptions({
		queryKey: ['catalog-car', id],
		queryFn: ({ signal }) => fetchCar(id, signal),
		refetchOnWindowFocus: false
	});
}
