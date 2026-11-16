import { fetchHomeBanners, fetchHomeFeaturedCars } from '#lib/api/catalog/public.server.ts';
import { toCarPreview } from '#lib/utils/car-preview.ts';
import type { PageServerLoad } from './$types';

export const load = (async () => {
	const [banners, featured] = await Promise.allSettled([
		fetchHomeBanners(),
		fetchHomeFeaturedCars()
	]);

	return {
		banners: banners.status === 'fulfilled' ? banners.value : [],
		featuredCars: featured.status === 'fulfilled' ? featured.value.map(toCarPreview) : [],
		featuredUnavailable: featured.status === 'rejected'
	};
}) satisfies PageServerLoad;
