import { describe, expect, it } from 'vitest';
import {
	bookingHistorySearch,
	readBookingFilters,
	bookingPageCount,
	bookingStatuses
} from './booking-history.ts';

describe('booking history URL state', () => {
	it('round trips a backend status and page', () => {
		const search = bookingHistorySearch('payment_review', 2);
		expect(search).toBe('?status=payment_review&page=2');
		expect(readBookingFilters(new URLSearchParams(search))).toEqual({
			status: 'payment_review',
			page: 2
		});
		expect(bookingHistorySearch('all')).toBe('');
		expect(bookingHistorySearch('confirmed')).toBe('?status=confirmed');
	});
	it.each(['0', '-1', '1e2', 'Infinity', '9007199254740992', 'bad'])(
		'normalizes invalid page %s',
		(page) => {
			expect(readBookingFilters(new URLSearchParams({ status: 'toString', page }))).toEqual({
				status: 'all',
				page: 1
			});
		}
	);
	it.each(['pending_payment', 'confirmed', 'expired', 'payment_review'])(
		'accepts backend status %s',
		(status) => {
			expect(readBookingFilters(new URLSearchParams({ status })).status).toBe(status);
		}
	);
	it('uses five rows per page, including the empty and partial last page', () => {
		expect([0, 1, 5, 6, 10, 11].map(bookingPageCount)).toEqual([1, 1, 1, 2, 2, 3]);
	});
	it('maps all backend statuses to distinct semantic tones', () => {
		expect(Object.values(bookingStatuses).map((status) => status.label)).toEqual([
			'Pending payment',
			'Confirmed',
			'Expired',
			'Under review'
		]);
		expect(new Set(Object.values(bookingStatuses).map((status) => status.tone)).size).toBe(4);
	});
});
