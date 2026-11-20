import { beforeEach, describe, expect, it, vi } from 'vitest';
vi.mock('$app/env/public', () => ({ PUBLIC_GATEWAY_BASE_URL: 'http://localhost:8080' }));
import { customerApi } from './customer.ts';
import { registerCustomer } from './identity/customer-auth.ts';
import { createBooking, fetchRentalQuote } from './booking.ts';
import { createVirtualAccount, refreshCustomerPayment, fetchCustomerInvoice } from './payment.ts';
import type { HoldInput } from '#lib/types/booking.ts';
let requests: { url: string; body: unknown; key: unknown; credentials: boolean | undefined }[];
beforeEach(() => {
	requests = [];
	customerApi.defaults.adapter = async (config) => {
		requests.push({
			url: config.url!,
			body: typeof config.data === 'string' ? JSON.parse(config.data) : config.data,
			key: config.headers.get('Idempotency-Key'),
			credentials: config.withCredentials
		});
		return {
			status: 202,
			statusText: 'Accepted',
			headers: {},
			config,
			data: { data: { status: 'initiating', va_number: null } }
		};
	};
});
describe('customer rental API', () => {
	it('sends the agreed payload and idempotency keys with cookie credentials', async () => {
		const input: HoldInput = {
			variant_id: 'car',
			pickup_at: '2026-11-24T08:00:00+08:00',
			return_at: '2026-11-24T18:00:00+08:00',
			expected_total_idr: 500000,
			terms_version: 'rental-terms-v1',
			accept_terms: true
		};
		await createBooking(input, 'hold-key');
		const result = await createVirtualAccount('booking', 'payment-key');
		await refreshCustomerPayment('booking');
		expect(requests).toEqual([
			{ url: '/api/v1/customer/bookings', body: input, key: 'hold-key', credentials: true },
			{
				url: '/api/v1/customer/bookings/booking/payments/va',
				body: {},
				key: 'payment-key',
				credentials: true
			},
			{
				url: '/api/v1/customer/bookings/booking/payment/refresh',
				body: {},
				key: undefined,
				credentials: true
			}
		]);
		expect(result).toEqual({ status: 'initiating', va_number: null });
	});
	it('registers through public Identity without requesting cookie credentials', async () => {
		await registerCustomer({ email: 'rental@example.invalid', password: 'test-password' });
		expect(requests[0].url).toBe('/api/v1/identity/register');
		expect(requests[0].credentials).toBe(false);
	});
	it('passes query cancellation and preserves structured API failures', async () => {
		const abort = new AbortController();
		let signal: unknown;
		const failure = {
			response: { status: 409, data: { error: { code: 'INVOICE_NOT_AVAILABLE' } } }
		};
		customerApi.defaults.adapter = async (config) => {
			signal = config.signal;
			throw failure;
		};
		await expect(
			fetchRentalQuote(
				{ variant_id: 'car', pickup_at: 'pickup', return_at: 'return' },
				abort.signal
			)
		).rejects.toBe(failure);
		expect(signal).toBe(abort.signal);
		await expect(fetchCustomerInvoice('booking')).rejects.toBe(failure);
	});
});
