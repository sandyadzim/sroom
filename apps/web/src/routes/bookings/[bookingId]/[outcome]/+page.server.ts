import { error } from '@sveltejs/kit';
import { isBookingPreviewId } from '#lib/utils/booking-preview.ts';
import type { PageServerLoad } from './$types';

export const load = (({ params }) => {
	if (
		!import.meta.env.DEV ||
		!isBookingPreviewId(params.bookingId) ||
		(params.outcome !== 'success' && params.outcome !== 'expired' && params.outcome !== 'review')
	) {
		error(404, 'Booking status is unavailable.');
	}
	return { bookingId: params.bookingId, outcome: params.outcome };
}) satisfies PageServerLoad;
