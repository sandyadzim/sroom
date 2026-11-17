import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	addBookingHistorySamples,
	bookingOutcomeFixture,
	bookingOutcomeRoute,
	checkBookingPreview,
	isBookingPreviewId,
	listBookingPreviews,
	readBookingPreview,
	remainingPaymentSeconds,
	saveBookingPreview
} from './booking-preview.ts';

afterEach(() => {
	vi.unstubAllGlobals();
	vi.useRealTimers();
});

const quote = {
	carName: 'Toyota Fortuner',
	carTrim: '2.8 VRZ',
	pickupAt: '2026-11-18T09:00:00+08:00',
	returnAt: '2026-11-20T17:00:00+08:00',
	days: 3,
	dailyRate: 850_000,
	total: 2_550_000
};

function usePreviewStorage() {
	const storage = new Map<string, string>();
	vi.stubGlobal('sessionStorage', {
		getItem: (key: string) => storage.get(key),
		get length() {
			return storage.size;
		},
		key: (index: number) => [...storage.keys()][index] ?? null,
		setItem: (key: string, value: string) => storage.set(key, value)
	});
}

describe('checkout handoff to payment', () => {
	it('keeps the original schedule, amount, and deadline when reloaded', () => {
		usePreviewStorage();
		vi.useFakeTimers();
		const start = new Date('2026-11-17T12:00:00+08:00').getTime();
		vi.setSystemTime(start);
		const id = saveBookingPreview(quote);
		expect(isBookingPreviewId(id)).toBe(true);
		vi.setSystemTime(start + 60_000);
		expect(readBookingPreview(id)).toEqual({
			...quote,
			createdAt: start,
			expiresAt: start + 900_000
		});
		expect(readBookingPreview('missing')).toBeNull();
	});

	it.each([
		['confirmed', 'success'],
		['expired', 'expired'],
		['review', 'review']
	] as const)('records the %s outcome only after a status check', (outcome, route) => {
		usePreviewStorage();
		const id = saveBookingPreview(quote, outcome);
		expect(readBookingPreview(id)?.outcome).toBeUndefined();
		expect(checkBookingPreview(id)).toBe(outcome);
		expect(readBookingPreview(id)?.outcome).toBe(outcome);
		expect(bookingOutcomeRoute(outcome)).toBe(route);
		expect(checkBookingPreview(id)).toBe(outcome);
	});

	it('leaves an ordinary preview pending when its deadline passes', () => {
		usePreviewStorage();
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-11-17T12:00:00+08:00'));
		const id = saveBookingPreview(quote, bookingOutcomeFixture('success'));
		vi.advanceTimersByTime(900_000);
		expect(checkBookingPreview(id)).toBeUndefined();
		expect(readBookingPreview(id)?.outcome).toBeUndefined();
	});

	it('rounds up remaining seconds and clamps at zero after the deadline', () => {
		expect(remainingPaymentSeconds(2000, 1999)).toBe(1);
		expect(remainingPaymentSeconds(2000, 2000)).toBe(0);
		expect(remainingPaymentSeconds(2000, 2001)).toBe(0);
	});
});

describe('booking history preview', () => {
	it('preserves local reservations and adds each sample only once', () => {
		usePreviewStorage();
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-11-17T12:00:00+08:00'));
		addBookingHistorySamples();
		vi.advanceTimersByTime(60_000);
		const id = saveBookingPreview(quote, 'confirmed');
		checkBookingPreview(id);
		addBookingHistorySamples();

		const bookings = listBookingPreviews();
		expect(bookings).toHaveLength(5);
		expect(bookings[0]).toMatchObject({ id, outcome: 'confirmed', total: quote.total });
		expect(bookings.map((booking) => booking.outcome)).toEqual([
			'confirmed',
			undefined,
			'confirmed',
			'expired',
			'review'
		]);
	});
});
