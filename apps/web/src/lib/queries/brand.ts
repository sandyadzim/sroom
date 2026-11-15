import { queryOptions } from '@tanstack/svelte-query';
import { brandsEndpoint, fetchBrandsForSelect } from '#lib/api/catalog/brand.ts';

export const brandOptionsQuery = queryOptions({
	queryKey: [brandsEndpoint, 'options'],
	queryFn: ({ signal }) => fetchBrandsForSelect(signal)
});
