import type { CarPreview } from '#lib/types/car.ts';

type FixtureCar = CarPreview & { image: string };

// Illustrative fixtures carried over from the local prototype, not live inventory or quotes.
const baseCatalogCars: FixtureCar[] = [
	{
		id: 'innova-zenix-v-cvt',
		brand: 'Toyota',
		model: 'Innova Zenix',
		trim: '2.0 V CVT',
		seats: 7,
		transmission: 'Automatic',
		dailyRate: 750000,
		image: '/images/fleet/innova.webp'
	},
	{
		id: 'fortuner-vrz-at',
		brand: 'Toyota',
		model: 'Fortuner',
		trim: '2.8 VRZ AT',
		seats: 7,
		transmission: 'Automatic',
		dailyRate: 950000,
		image: '/images/fleet/fortuner.webp'
	},
	{
		id: 'brio-rs-cvt',
		brand: 'Honda',
		model: 'Brio',
		trim: 'RS CVT',
		seats: 5,
		transmission: 'Automatic',
		dailyRate: 300000,
		image: '/images/fleet/brio.webp'
	}
];

// Eight display-only variants for exercising the six-per-page catalogue slice.
export const demoCatalogCars: FixtureCar[] = [
	...baseCatalogCars,
	{ ...baseCatalogCars[2], id: 'brio-satya-e-cvt', trim: 'Satya E CVT', dailyRate: 275000 },
	{ ...baseCatalogCars[0], id: 'innova-zenix-g-cvt', trim: '2.0 G CVT', dailyRate: 650000 },
	{
		...baseCatalogCars[1],
		id: 'fortuner-gr-sport-at',
		trim: '2.8 GR Sport AT',
		dailyRate: 1100000
	},
	{
		id: 'jimny-3-door-at',
		brand: 'Suzuki',
		model: 'Jimny',
		trim: '3-door AT',
		seats: 4,
		transmission: 'Automatic',
		dailyRate: 800000,
		image: '/images/hero/jimny-studio.webp'
	},
	{
		id: 'jimny-5-door-at',
		brand: 'Suzuki',
		model: 'Jimny',
		trim: '5-door AT',
		seats: 4,
		transmission: 'Automatic',
		dailyRate: 900000,
		image: '/images/hero/jimny-studio.webp'
	}
];
