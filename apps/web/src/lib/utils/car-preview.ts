import { resolve } from '$app/paths';
import type { CarPreview } from '#lib/types/car.ts';
import type { CatalogCarCard } from '#lib/types/catalog.ts';
import { carCategories, carOptionLabel, carTransmissions } from './car.ts';

export function toCarPreview(car: CatalogCarCard): CarPreview {
	return {
		id: car.id,
		brand: car.brand_name,
		model: car.model,
		trim: car.trim,
		category: carOptionLabel(carCategories, car.category),
		seats: car.seats,
		transmission: carOptionLabel(carTransmissions, car.transmission),
		dailyRate: car.daily_rate_idr,
		image: car.cover?.url ?? null,
		imageAlt: car.cover?.alt_text,
		href: resolve('/cars/[slug]', { slug: car.slug })
	};
}
