import { error } from '@sveltejs/kit';
import { isBookingPreviewId } from '#lib/utils/booking-preview.ts';
import type { PageServerLoad } from './$types';

export const load = (({ params }) => {
	if (!isBookingPreviewId(params.bookingId)) error(404, 'Invoice is unavailable.');
	return { bookingId: params.bookingId };
}) satisfies PageServerLoad;
