import { describe, expect, it } from 'vitest';
import type {
	BookingStatusHistoryEntry,
	PaymentTimelineEntry,
	StaffPayment
} from '#lib/types/admin-transactions.ts';
import { paymentStatusStyles } from './admin-transaction-status.ts';
import {
	attachCustomerNames,
	bookingPaymentAttemptParams,
	bookingTimeline,
	classifyInvoiceFailure,
	customerDisplayName,
	isBookingStatus,
	isStaffVariantId,
	matchingBookingPayments,
	paymentTimeline,
	uniqueCustomerIds
} from './staff-transactions.ts';

const bookingId = '11111111-1111-4111-8111-111111111111';
const otherBookingId = '22222222-2222-4222-8222-222222222222';

function payment(id: string, booking: string): StaffPayment {
	return {
		payment_id: id,
		booking_id: booking,
		customer_id: '33333333-3333-4333-8333-333333333333',
		provider: 'midtrans',
		provider_order_id: `order-${id}`,
		provider_transaction_id: null,
		status: 'pending',
		bank: 'bni',
		amount_idr: 1_500_000,
		currency: 'IDR',
		review_required: false,
		review_reason: null,
		hold_expires_at: '2026-11-20T03:15:00Z',
		expiry_review_required: false,
		created_at: '2026-11-20T03:00:00Z',
		updated_at: '2026-11-20T03:00:00Z'
	};
}

describe('staff booking timeline adapter', () => {
	it('maps created and transition history without inventing events', () => {
		const history: BookingStatusHistoryEntry[] = [
			{
				id: 'history-1',
				from_status: null,
				to_status: 'pending_payment',
				actor_type: 'customer',
				reason_code: null,
				occurred_at: '2026-11-20T03:00:00Z'
			},
			{
				id: 'history-2',
				from_status: 'pending_payment',
				to_status: 'confirmed',
				actor_type: 'provider_event',
				reason_code: 'payment.verified',
				occurred_at: '2026-11-20T03:05:00Z'
			}
		];

		const events = bookingTimeline(history);

		expect(events).toHaveLength(2);
		expect(events[0]).toMatchObject({
			title: 'Booking created',
			time: '2026-11-20T03:00:00Z',
			description: 'Actor: Customer'
		});
		expect(events[1]).toMatchObject({
			title: 'Status changed to Confirmed',
			time: '2026-11-20T03:05:00Z',
			description: 'Actor: Provider event · Reason: payment.verified'
		});
	});

	it('returns no timeline for an empty history', () => {
		expect(bookingTimeline([])).toEqual([]);
	});
});

describe('staff payment timeline adapter', () => {
	it('maps safe observation fields and keeps backend order', () => {
		const entries: PaymentTimelineEntry[] = [
			{
				id: 'timeline-1',
				source: 'created',
				occurred_at: '2026-11-20T03:00:00Z',
				provider_status: null,
				verification_result: null,
				error_code: null
			},
			{
				id: 'timeline-2',
				source: 'status_lookup',
				occurred_at: '2026-11-20T03:05:00Z',
				provider_status: 'pending',
				verification_result: 'unavailable',
				error_code: 'PROVIDER_TIMEOUT'
			}
		];

		const events = paymentTimeline(entries);

		expect(events.map(({ title }) => title)).toEqual(['Payment started', 'Provider status check']);
		expect(events[1]).toMatchObject({
			time: '2026-11-20T03:05:00Z',
			description: 'Provider status: pending · Verification: unavailable · Error: PROVIDER_TIMEOUT'
		});
		expect(events[0]?.description).toBeUndefined();
	});
});

describe('staff booking payment lookup', () => {
	it('matches booking IDs exactly and ignores substring collisions', () => {
		const rows = [
			payment('payment-1', bookingId),
			payment('payment-2', otherBookingId),
			payment('payment-3', `${bookingId}-suffix`)
		];

		expect(matchingBookingPayments(rows, bookingId).map(({ payment_id }) => payment_id)).toEqual([
			'payment-1'
		]);
	});

	it('requests one bounded search page for the booking UUID', () => {
		expect(bookingPaymentAttemptParams(bookingId)).toEqual({
			page: 1,
			limit: 100,
			search: bookingId
		});
	});
});

describe('staff customer names', () => {
	it('uses the profile name, falls back to email, and keeps unknown IDs explicit', () => {
		expect(customerDisplayName({ full_name: 'Rani Putri', email: 'rani@example.test' })).toBe(
			'Rani Putri'
		);
		expect(customerDisplayName({ full_name: '   ', email: 'rani@example.test' })).toBe(
			'rani@example.test'
		);

		const rows = uniqueCustomerIds([{ customer_id: bookingId }, { customer_id: otherBookingId }]);
		expect(rows).toEqual([bookingId, otherBookingId]);

		const enriched = attachCustomerNames(
			[
				{ customer_id: bookingId, id: 'row-1' },
				{ customer_id: otherBookingId, id: 'row-2' }
			],
			[{ id: bookingId, full_name: 'Rani Putri', email: 'rani@example.test' }]
		);
		expect(enriched).toEqual([
			{ customer_id: bookingId, id: 'row-1', customer_name: 'Rani Putri' },
			{ customer_id: otherBookingId, id: 'row-2', customer_name: null }
		]);
	});
});

describe('staff filter and invoice helpers', () => {
	it('accepts only UUID vehicle filters', () => {
		expect(isStaffVariantId(bookingId)).toBe(true);
		expect(isStaffVariantId('Toyota Fortuner')).toBe(false);
		expect(isStaffVariantId('')).toBe(false);
	});

	it('classifies unavailable, missing and failed invoices separately', () => {
		expect(classifyInvoiceFailure(409, 'INVOICE_NOT_AVAILABLE')).toBe('not_available');
		expect(classifyInvoiceFailure(409, 'OTHER_CODE')).toBe('error');
		expect(classifyInvoiceFailure(404)).toBe('not_found');
		expect(classifyInvoiceFailure(503)).toBe('error');
		expect(classifyInvoiceFailure(undefined)).toBe('error');
	});

	it('recognises only contract booking statuses', () => {
		expect(isBookingStatus('payment_review')).toBe(true);
		expect(isBookingStatus('paid')).toBe(false);
	});

	it('labels the initiating payment status', () => {
		expect(paymentStatusStyles.initiating.label).toBe('Initiating');
	});
});
