import { resolve } from '$app/paths';
import type { CarDetail, CarPreview } from '#lib/types/car.ts';
import type { CatalogCarCard, CatalogCarDetail } from '#lib/types/catalog.ts';
import {
	carBodyTypes,
	carCategories,
	carFuelTypes,
	carOptionLabel,
	carTransmissions
} from './car.ts';

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

export function toCarDetail(car: CatalogCarDetail): CarDetail {
	return {
		...toCarPreview(car),
		slug: car.slug,
		status: car.status,
		bodyType: carOptionLabel(carBodyTypes, car.body_type),
		fuel: carOptionLabel(carFuelTypes, car.fuel_type),
		description: car.description,
		gallery: car.gallery.flatMap((photo) =>
			photo.url ? [{ src: photo.url, alt: photo.alt_text, label: photo.label }] : []
		)
	};
}
