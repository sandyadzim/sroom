import { queryOptions } from '@tanstack/svelte-query';
import { lookupCustomers } from '#lib/api/identity/customer.ts';
import {
	fetchBookingPaymentAttempts,
	fetchStaffBooking,
	fetchStaffInvoice,
	fetchStaffPayment
} from '#lib/api/staff-transactions.ts';

export const staffTransactionsKey = ['staff-transactions'] as const;

export function staffBookingQuery(id: string) {
	return queryOptions({
		queryKey: [...staffTransactionsKey, 'booking', id],
		queryFn: ({ signal }) => fetchStaffBooking(id, signal),
		retry: false,
		staleTime: 0
	});
}

export function staffPaymentQuery(id: string) {
	return queryOptions({
		queryKey: [...staffTransactionsKey, 'payment', id],
		queryFn: ({ signal }) => fetchStaffPayment(id, signal),
		retry: false,
		staleTime: 0
	});
}

export function staffInvoiceQuery(bookingId: string) {
	return queryOptions({
		queryKey: [...staffTransactionsKey, 'invoice', bookingId],
		queryFn: ({ signal }) => fetchStaffInvoice(bookingId, signal),
		retry: false,
		staleTime: 0
	});
}

export function bookingPaymentAttemptsQuery(bookingId: string) {
	return queryOptions({
		queryKey: [...staffTransactionsKey, 'booking-payments', bookingId],
		queryFn: ({ signal }) => fetchBookingPaymentAttempts(bookingId, signal),
		retry: false,
		staleTime: 0
	});
}

export function customerLookupQuery(ids: string[]) {
	const uniqueIds = [...new Set(ids)].sort();
	return queryOptions({
		queryKey: [...staffTransactionsKey, 'customers', uniqueIds],
		queryFn: ({ signal }) => lookupCustomers(uniqueIds, signal),
		enabled: uniqueIds.length > 0,
		retry: false,
		staleTime: 60_000
	});
}
