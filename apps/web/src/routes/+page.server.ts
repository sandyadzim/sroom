import { resolve } from '$app/paths';
import { fetchHomeBanners, fetchHomeFeaturedCars } from '#lib/api/catalog/home.server.ts';
import type { CarPreview } from '#lib/types/car.ts';
import { carOptionLabel, carTransmissions } from '#lib/utils/car.ts';
import type { PageServerLoad } from './$types';

export const load = (async () => {
	const [banners, featured] = await Promise.allSettled([
		fetchHomeBanners(),
		fetchHomeFeaturedCars()
	]);

	return {
		banners: banners.status === 'fulfilled' ? banners.value : [],
		featuredCars:
			featured.status === 'fulfilled'
				? featured.value.map((car): CarPreview => ({
						id: car.id,
						brand: car.brand_name,
						model: car.model,
						trim: car.trim,
						seats: car.seats,
						transmission: carOptionLabel(carTransmissions, car.transmission),
						dailyRate: car.daily_rate_idr,
						image: car.cover?.url ?? null,
						imageAlt: car.cover?.alt_text,
						href: resolve('/cars/[slug]', { slug: car.slug })
					}))
				: [],
		featuredUnavailable: featured.status === 'rejected'
	};
}) satisfies PageServerLoad;
