import { error } from '@sveltejs/kit';
import { isBookingPreviewId } from '#lib/utils/booking-preview.ts';
import type { PageServerLoad } from './$types';

export const load = (({ params, url }) => {
	if (!import.meta.env.DEV || !isBookingPreviewId(params.bookingId)) {
		error(404, 'Payment is unavailable.');
	}

	const requested = url.searchParams.get('scenario');
	const scenario =
		requested === 'va-error' || requested === 'status-error' || requested === 'deadline'
			? requested
			: 'pending';

	return { bookingId: params.bookingId, scenario };
}) satisfies PageServerLoad;
