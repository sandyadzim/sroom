import { queryOptions } from '@tanstack/svelte-query';
import { fetchBannerSlot, fetchBannerSlots } from '#lib/api/catalog/banner.ts';
import { brandsEndpoint, fetchBrandsForSelect } from '#lib/api/catalog/brand.ts';

export const bannerSlotsQuery = queryOptions({
	queryKey: ['banner-slots'],
	queryFn: ({ signal }) => fetchBannerSlots(signal)
});

export function bannerSlotQuery(position: number) {
	return queryOptions({
		queryKey: [...bannerSlotsQuery.queryKey, position],
		queryFn: ({ signal }) => fetchBannerSlot(position, signal),
		enabled: Number.isInteger(position) && position >= 1 && position <= 3
	});
}

export const bannerBrandsQuery = queryOptions({
	queryKey: [brandsEndpoint, 'options'],
	queryFn: ({ signal }) => fetchBrandsForSelect(signal)
});
