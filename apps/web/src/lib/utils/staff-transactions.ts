import { isAxiosError } from 'axios';
import type {
	BookingStatusHistoryEntry,
	PaymentTimelineEntry,
	StaffPayment,
	TransactionEvent
} from '#lib/types/admin-transactions.ts';
import type { ApiError, CustomerLookup } from '#lib/types/identity.ts';
import { bookingStatusStyles } from '#lib/utils/admin-transaction-status.ts';

const paymentSourceTitles: Record<PaymentTimelineEntry['source'], string> = {
	created: 'Payment started',
	charge: 'Provider charge',
	webhook: 'Provider notification',
	status_lookup: 'Provider status check'
};

const actorLabels: Record<BookingStatusHistoryEntry['actor_type'], string> = {
	system: 'System',
	customer: 'Customer',
	admin: 'Admin',
	provider_event: 'Provider event'
};

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export const bookingPaymentAttemptLimit = 100;

export function bookingPaymentAttemptParams(bookingId: string) {
	return { page: 1, limit: bookingPaymentAttemptLimit, search: bookingId };
}

export function isStaffVariantId(value: string) {
	return uuidPattern.test(value);
}

const bookingStatuses = ['pending_payment', 'confirmed', 'expired', 'payment_review'] as const;

export function isBookingStatus(value: string): value is (typeof bookingStatuses)[number] {
	return (bookingStatuses as readonly string[]).includes(value);
}

export function bookingTimeline(history: BookingStatusHistoryEntry[]): TransactionEvent[] {
	return history.map((entry) => {
		const title = entry.from_status
			? `Status changed to ${bookingStatusStyles[entry.to_status].label}`
			: 'Booking created';
		const details = [`Actor: ${actorLabels[entry.actor_type]}`];
		if (entry.reason_code) details.push(`Reason: ${entry.reason_code}`);
		return {
			title,
			time: entry.occurred_at,
			description: details.join(' · ')
		};
	});
}

export function paymentTimeline(entries: PaymentTimelineEntry[]): TransactionEvent[] {
	return entries.map((entry) => {
		const details: string[] = [];
		if (entry.provider_status) details.push(`Provider status: ${entry.provider_status}`);
		if (entry.verification_result) details.push(`Verification: ${entry.verification_result}`);
		if (entry.error_code) details.push(`Error: ${entry.error_code}`);
		return {
			title: paymentSourceTitles[entry.source],
			time: entry.occurred_at,
			...(details.length ? { description: details.join(' · ') } : {})
		};
	});
}

/**
 * The Booking detail route has no payment ID, so the payment panel searches
 * Payment by booking ID and keeps only rows whose booking_id matches exactly.
 */
export function matchingBookingPayments(payments: StaffPayment[], bookingId: string) {
	return payments.filter((payment) => payment.booking_id === bookingId);
}

export function customerDisplayName(customer: Pick<CustomerLookup, 'full_name' | 'email'>) {
	return customer.full_name.trim() || customer.email;
}

export function uniqueCustomerIds(rows: readonly { customer_id: string }[]) {
	return [...new Set(rows.map((row) => row.customer_id))];
}

export function attachCustomerNames<T extends { customer_id: string }>(
	rows: T[],
	customers: CustomerLookup[]
): (T & { customer_name: string | null })[] {
	const names = new Map(customers.map((customer) => [customer.id, customerDisplayName(customer)]));
	return rows.map((row) => ({
		...row,
		customer_name: names.get(row.customer_id) ?? null
	}));
}

export type InvoiceFailure = 'not_available' | 'not_found' | 'error';

export function classifyInvoiceFailure(status?: number, code?: string): InvoiceFailure {
	if (status === 409 && code === 'INVOICE_NOT_AVAILABLE') return 'not_available';
	if (status === 404) return 'not_found';
	return 'error';
}

export function invoiceFailure(cause: unknown): InvoiceFailure {
	if (!isAxiosError<ApiError>(cause)) return 'error';
	return classifyInvoiceFailure(cause.response?.status, cause.response?.data?.error?.code);
}
