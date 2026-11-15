import { queryOptions } from '@tanstack/svelte-query';
import { fetchBannerSlot, fetchBannerSlots } from '#lib/api/catalog/banner.ts';

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
