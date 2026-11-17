export const bookingStatuses = {
	pending: { label: 'Pending payment', tone: 'bg-caution/10 text-caution' },
	confirmed: { label: 'Confirmed', tone: 'bg-positive/10 text-positive' },
	expired: { label: 'Expired', tone: 'bg-panel-muted text-ink-muted' },
	review: { label: 'Under review', tone: 'bg-information/10 text-information' }
};

export type BookingStatus = keyof typeof bookingStatuses;
type BookingFilter = BookingStatus | 'all';

export function readBookingFilters(params: Pick<URLSearchParams, 'get'>) {
	const value = params.get('status');
	const status: BookingFilter =
		value && Object.hasOwn(bookingStatuses, value) ? (value as BookingStatus) : 'all';
	const rawPage = params.get('page') ?? '';
	const page = /^[1-9]\d*$/.test(rawPage) ? Number(rawPage) : 1;
	return { status, page: Number.isSafeInteger(page) && page > 0 ? page : 1 };
}

export function bookingHistorySearch(status: BookingFilter, page = 1) {
	const params = new URLSearchParams();
	if (status !== 'all') params.set('status', status);
	if (page > 1) params.set('page', String(page));
	return params.size ? (`?${params}` as const) : '';
}
