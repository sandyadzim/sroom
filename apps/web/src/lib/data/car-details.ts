import { demoCatalogCars } from './demo-fleet.ts';
import type { CarDetail, CarGalleryImage } from '#lib/types/car.ts';

// Display content only. Checkout prices and physical-unit allocation remain server-owned.
const descriptions = {
	'Innova Zenix': {
		category: 'MPV',
		fuel: 'Petrol',
		description:
			'Family plans, a weekend together, or a longer route home. The Innova Zenix pairs seven seats with an automatic transmission, giving your day a little more room.',
		studio: undefined
	},
	Fortuner: {
		category: 'SUV',
		fuel: 'Diesel',
		description:
			'A spacious seven-seat SUV for family days and trips with more on the agenda. The Fortuner brings a bold presence to the road, with an automatic transmission for an easygoing drive.',
		studio: '/images/hero/fortuner-studio.webp'
	},
	Brio: {
		category: 'Hatchback',
		fuel: 'Petrol',
		description:
			'Coffee stops, everyday errands, and a change of plans. The compact Honda Brio is a natural fit for city days, with five seats and an automatic transmission to keep things simple.',
		studio: '/images/hero/brio-studio.webp'
	},
	Jimny: {
		category: 'SUV',
		fuel: 'Petrol',
		description:
			'A distinctive boxy silhouette for a different kind of day. The Suzuki Jimny pairs four seats with an automatic transmission, making room for a small group and a spontaneous city detour.',
		studio: undefined
	}
};

export function getCarDetail(slug: string): CarDetail | undefined {
	const car = demoCatalogCars.find((item) => item.id === slug);
	if (!car) return undefined;
	const content = descriptions[car.model as keyof typeof descriptions];
	const gallery: CarGalleryImage[] = [
		{
			src: car.image,
			alt: `Illustrative ${car.brand} ${car.model} exterior`,
			label: 'Exterior'
		}
	];
	if (content.studio) {
		gallery.push({
			src: content.studio,
			alt: `Illustrative ${car.brand} ${car.model} studio composition`,
			label: 'Studio'
		});
	}
	return { ...car, ...content, gallery };
}

export function getRelatedCars(slug: string) {
	const selected = demoCatalogCars.find((car) => car.id === slug);
	return demoCatalogCars
		.filter(
			(car, index, cars) =>
				car.model !== selected?.model &&
				cars.findIndex((item) => item.model === car.model) === index
		)
		.slice(0, 3);
}
