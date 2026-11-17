import { describe, expect, it } from 'vitest';
import {
	findBookingPreview,
	findPaymentPreview,
	listAdminBookingPreviews,
	listAdminPaymentPreviews
} from './admin-transaction-preview.ts';

describe('admin transaction fixtures', () => {
	it('filters bookings by vehicle, status and local pickup day', () => {
		const result = listAdminBookingPreviews({
			page: 1,
			limit: 20,
			filters: {
				model: 'Toyota Fortuner',
				status: 'confirmed',
				pickup_date: '2026-11-20',
				search: 'alex'
			}
		});

		expect(result.page.total).toBe(1);
		expect(result.data[0]?.id).toBe('BKG-202611-024');
	});

	it('keeps the filtered total while paginating', () => {
		const result = listAdminBookingPreviews({ page: 2, limit: 20, filters: {} });
		expect(result.page.total).toBe(24);
		expect(result.data.map(({ id }) => id)).toEqual([
			'BKG-202611-004',
			'BKG-202611-003',
			'BKG-202611-002',
			'BKG-202611-001'
		]);
	});

	it('keeps verified payment separate from booking review after release', () => {
		const result = listAdminPaymentPreviews({
			page: 1,
			limit: 20,
			filters: { status: 'paid', reconciliation: 'required', search: 'BKG-202611-023' }
		});
		const payment = result.data[0];

		expect(result.page.total).toBe(1);
		expect(payment?.status).toBe('paid');
		expect(payment?.providerReference).toMatch(/^•••• [0-9]{4}$/);
		const booking = findPaymentPreview(payment.id)?.booking;
		expect(booking?.status).toBe('payment_review');
		expect(booking?.unit).toBeNull();
		expect(booking?.timeline.map(({ title }) => title)).toContain('Reservation released');
	});

	it('supports bookings without a payment attempt', () => {
		expect(findBookingPreview('BKG-202611-002')?.payment).toBeNull();
		expect(findPaymentPreview('PAY-202611-002')).toBeUndefined();
	});
});
