import { error } from '@sveltejs/kit';
import { earliestPickupDate } from './schedule.ts';
import type { PageServerLoad } from './$types';

export const load = (({ params, url }) => {
	if (!import.meta.env.DEV || params.slug !== 'toyota-fortuner') {
		error(404, 'Checkout is unavailable.');
	}

	const requested = url.searchParams.get('scenario');
	const scenario =
		requested === 'unavailable' || requested === 'error' || requested === 'conflict'
			? requested
			: 'available';

	return {
		car: {
			name: 'Toyota Fortuner',
			trim: '2.8 VRZ',
			category: 'SUV',
			image: 'images/hero/fortuner-studio.webp' as const,
			dailyRate: 850_000
		},
		customer: { name: 'Alex Morgan', email: 'alex@example.invalid' },
		earliestPickup: earliestPickupDate(),
		scenario
	};
}) satisfies PageServerLoad;
