import type { CarPreview } from '#lib/types/car.ts';

// Illustrative fixtures carried over from the local prototype, not live inventory or quotes.
export const demoFeaturedCars: CarPreview[] = [
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
