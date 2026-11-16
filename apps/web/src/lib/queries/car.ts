import { queryOptions } from '@tanstack/svelte-query';
import { carsEndpoint, fetchCar, fetchCarsForSelect } from '#lib/api/catalog/car.ts';

export const carOptionsQuery = queryOptions({
	queryKey: [carsEndpoint, 'options'],
	queryFn: ({ signal }) => fetchCarsForSelect({}, signal)
});

export function carSearchOptionsQuery(search: string, activeOnly: boolean) {
	if (activeOnly) return activeCarOptionsQuery(search);
	return queryOptions({
		queryKey: [...carOptionsQuery.queryKey, 'search', search],
		queryFn: ({ signal }) => fetchCarsForSelect({ search }, signal)
	});
}

export function activeCarOptionsQuery(search = '') {
	return queryOptions({
		queryKey: [carsEndpoint, 'options', 'active', search],
		queryFn: ({ signal }) => fetchCarsForSelect({ search, status: 'active' }, signal)
	});
}

export function carDetailQuery(id: string) {
	return queryOptions({
		queryKey: ['catalog-car', id],
		queryFn: ({ signal }) => fetchCar(id, signal),
		refetchOnWindowFocus: false
	});
}
