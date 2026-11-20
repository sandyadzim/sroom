import { error } from '@sveltejs/kit';
import { isBookingId } from '#lib/utils/customer-auth.ts';
import type { PageLoad } from './$types';
export const load = (({ params }) => {
	if (!isBookingId(params.bookingId) || !['success', 'expired', 'review'].includes(params.outcome))
		error(404, 'Booking is unavailable.');
	return { bookingId: params.bookingId, outcome: params.outcome };
}) satisfies PageLoad;
