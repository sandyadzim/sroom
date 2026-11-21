import { beforeEach, describe, expect, it, vi } from 'vitest';
vi.mock('$app/env/public', () => ({ PUBLIC_GATEWAY_BASE_URL: 'http://localhost:8080' }));
import { customerApi } from './customer.ts';
import { registerCustomer } from './identity/customer-auth.ts';
import { createBooking, fetchRentalQuote, fetchCustomerBookings } from './booking.ts';
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
	it('requests one backend page and preserves the server total and order', async () => {
		const result = {
			data: [{ id: 'newer' }, { id: 'older' }],
			page: { number: 2, limit: 5, total: 11 }
		};
		const controller = new AbortController();
		let input: unknown;
		customerApi.defaults.adapter = async (config) => {
			input = {
				url: config.url,
				params: config.params,
				signal: config.signal,
				credentials: config.withCredentials
			};
			return { data: result, status: 200, statusText: 'OK', headers: {}, config };
		};
		expect(
			await fetchCustomerBookings({ page: 2, limit: 5, status: 'confirmed' }, controller.signal)
		).toEqual(result);
		expect(input).toEqual({
			url: '/api/v1/customer/bookings',
			params: { page: 2, limit: 5, status: 'confirmed' },
			signal: controller.signal,
			credentials: true
		});
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
