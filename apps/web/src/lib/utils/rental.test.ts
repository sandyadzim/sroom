import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import {
	canShowVirtualAccount,
	holdCommand,
	paymentDeadline,
	readPendingHold,
	rentalOutcome,
	virtualAccountKey
} from './rental.ts';
import type { Booking, HoldInput } from '#lib/types/booking.ts';
import type { CustomerPayment } from '#lib/types/payment.ts';
const payload: HoldInput = {
	variant_id: 'car',
	pickup_at: '2026-11-24T08:00:00+08:00',
	return_at: '2026-11-26T18:00:00+08:00',
	expected_total_idr: 1500000,
	terms_version: 'rental-terms-v1',
	accept_terms: true
};
const payment: CustomerPayment = {
	payment_id: 'payment',
	booking_id: 'booking',
	booking_status: 'pending_payment',
	status: 'pending',
	amount_idr: 1500000,
	currency: 'IDR',
	bank: 'bni',
	va_number: 'test-instructions',
	hold_expires_at: '2026-11-20T01:15:00Z',
	provider_expires_at: '2026-11-20T01:14:00Z',
	verified_at: null,
	provider_paid_at: null,
	review_required: false
};
beforeEach(() => {
	const storage = new Map<string, string>();
	vi.stubGlobal('sessionStorage', {
		getItem: (key: string) => storage.get(key) ?? null,
		setItem: (key: string, value: string) => storage.set(key, value),
		removeItem: (key: string) => storage.delete(key)
	});
});
afterEach(() => vi.unstubAllGlobals());
describe('rental retry commands', () => {
	it('reuses a hold key across reloads and separates customers, cars, and changed payloads', () => {
		const command = holdCommand('a', 'car', payload);
		expect(readPendingHold('a', 'car')).toEqual(command);
		expect(holdCommand('a', 'car', { ...payload }).key).toBe(command.key);
		expect(holdCommand('a', 'car', { ...payload, expected_total_idr: 1600000 }).key).not.toBe(
			command.key
		);
		expect(holdCommand('b', 'car', payload).key).not.toBe(command.key);
		expect(holdCommand('a', 'other', payload).key).not.toBe(command.key);
	});
	it('keeps a VA key per customer and booking', () => {
		const key = virtualAccountKey('a', 'booking');
		expect(virtualAccountKey('a', 'booking')).toBe(key);
		expect(virtualAccountKey('b', 'booking')).not.toBe(key);
		expect(virtualAccountKey('a', 'other')).not.toBe(key);
	});
	it('ignores damaged retry storage', () => {
		sessionStorage.setItem('sroom-rental:a:hold:car', '{');
		expect(readPendingHold('a', 'car')).toBeUndefined();
	});
});
describe('authoritative payment and booking states', () => {
	it('uses the earliest instruction deadline and hides instructions at the exact boundary', () => {
		const deadline = Date.parse(payment.provider_expires_at!);
		expect(paymentDeadline(payment)).toBe(deadline);
		expect(canShowVirtualAccount(payment, deadline - 1)).toBe(true);
		expect(canShowVirtualAccount(payment, deadline)).toBe(false);
		expect(canShowVirtualAccount({ ...payment, review_required: true }, deadline - 1)).toBe(false);
		expect(canShowVirtualAccount({ ...payment, status: 'initiating' }, deadline - 1)).toBe(false);
		expect(canShowVirtualAccount({ ...payment, va_number: null }, deadline - 1)).toBe(false);
	});
	it('keeps a paid but unconfirmed booking in processing and never derives expiry from the clock', () => {
		const booking = { status: 'pending_payment' } as Booking;
		expect(rentalOutcome(booking, { ...payment, status: 'paid' })).toBeUndefined();
		expect(rentalOutcome(booking, { ...payment, status: 'expired' })).toBeUndefined();
		expect(rentalOutcome(booking, { ...payment, review_required: true })).toBe('review');
		expect(rentalOutcome({ ...booking, status: 'confirmed' }, payment)).toBe('success');
		expect(rentalOutcome({ ...booking, status: 'expired' }, payment)).toBe('expired');
	});
});
