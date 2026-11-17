import { describe, expect, it } from 'vitest';
import { bookingHistorySearch, readBookingFilters } from './booking-history.ts';

describe('booking history URL state', () => {
	it('round trips a status and page', () => {
		const search = bookingHistorySearch('review', 2);
		expect(search).toBe('?status=review&page=2');
		expect(readBookingFilters(new URLSearchParams(search))).toEqual({ status: 'review', page: 2 });
	});

	it('accepts only defined statuses and ordinary positive page numbers', () => {
		expect(readBookingFilters(new URLSearchParams('status=toString&page=1e2'))).toEqual({
			status: 'all',
			page: 1
		});
		expect(bookingHistorySearch('all')).toBe('');
	});
});
