import { error } from '@sveltejs/kit';
import { bookingOutcomeFixture, checkoutPreviewSlug } from '#lib/utils/booking-preview.ts';
import { earliestPickupDate } from './schedule.ts';
import type { PageServerLoad } from './$types';

export const load = (({ params, url }) => {
	if (!import.meta.env.DEV || params.slug !== checkoutPreviewSlug) {
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
			dailyRate: 850_000
		},
		customer: { name: 'Alex Morgan', email: 'alex@example.invalid' },
		earliestPickup: earliestPickupDate(),
		nextOutcome: bookingOutcomeFixture(url.searchParams.get('outcome')),
		scenario
	};
}) satisfies PageServerLoad;
