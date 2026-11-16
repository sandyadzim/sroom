import { queryOptions } from '@tanstack/svelte-query';
import { fetchFeaturedCars } from '#lib/api/catalog/featured-car.ts';

export const featuredCarsQuery = queryOptions({
	queryKey: ['catalog-featured-cars'],
	queryFn: ({ signal }) => fetchFeaturedCars(signal),
	refetchOnWindowFocus: false
});
