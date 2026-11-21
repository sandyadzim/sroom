import type { BookingHistoryFilters, BookingStatus } from '#lib/types/booking.ts';

export const bookingPageSize = 5;
export const bookingStatuses = {
	pending_payment: { label: 'Pending payment', tone: 'warning' },
	confirmed: { label: 'Confirmed', tone: 'success' },
	expired: { label: 'Expired', tone: 'danger' },
	payment_review: { label: 'Under review', tone: 'info' }
} as const;

export function readBookingFilters(params: Pick<URLSearchParams, 'get'>): BookingHistoryFilters {
	const value = params.get('status');
	const status = value && Object.hasOwn(bookingStatuses, value) ? (value as BookingStatus) : 'all';
	const rawPage = params.get('page') ?? '';
	const page = /^[1-9]\d*$/.test(rawPage) ? Number(rawPage) : 1;
	return { status, page: Number.isSafeInteger(page) && page > 0 ? page : 1 };
}

export function bookingHistorySearch(status: BookingHistoryFilters['status'], page = 1) {
	const params = new URLSearchParams();
	if (status !== 'all') params.set('status', status);
	if (page > 1) params.set('page', String(page));
	return params.size ? (`?${params}` as const) : '';
}

export function bookingPageCount(total: number) {
	return Math.max(1, Math.ceil(total / bookingPageSize));
}
