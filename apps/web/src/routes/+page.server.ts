import {
	fetchHomeBanners,
	fetchHomeFeaturedCars,
	fetchPublicBrands
} from '#lib/api/catalog/public.server.ts';
import { toCarPreview } from '#lib/utils/car-preview.ts';
import type { PageServerLoad } from './$types';

export const load = (async () => {
	const [banners, featured, brands] = await Promise.allSettled([
		fetchHomeBanners(),
		fetchHomeFeaturedCars(),
		fetchPublicBrands(true)
	]);

	return {
		banners: banners.status === 'fulfilled' ? banners.value : [],
		featuredCars: featured.status === 'fulfilled' ? featured.value.map(toCarPreview) : [],
		featuredUnavailable: featured.status === 'rejected',
		brands: brands.status === 'fulfilled' ? brands.value : []
	};
}) satisfies PageServerLoad;
