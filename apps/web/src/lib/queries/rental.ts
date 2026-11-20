import { queryOptions } from '@tanstack/svelte-query';
import { fetchBooking } from '#lib/api/booking.ts';
import { fetchCustomerInvoice, fetchCustomerPayment } from '#lib/api/payment.ts';

export const customerRentalKey = ['customer-rental'] as const;
export function bookingQuery(customerId: string, bookingId: string) {
	return queryOptions({
		queryKey: [...customerRentalKey, customerId, 'booking', bookingId],
		queryFn: ({ signal }) => fetchBooking(bookingId, signal),
		retry: false,
		staleTime: 0
	});
}
export function paymentQuery(customerId: string, bookingId: string) {
	return queryOptions({
		queryKey: [...customerRentalKey, customerId, 'payment', bookingId],
		queryFn: ({ signal }) => fetchCustomerPayment(bookingId, signal),
		retry: false,
		staleTime: 0
	});
}
export function invoiceQuery(customerId: string, bookingId: string) {
	return queryOptions({
		queryKey: [...customerRentalKey, customerId, 'invoice', bookingId],
		queryFn: ({ signal }) => fetchCustomerInvoice(bookingId, signal),
		retry: false,
		staleTime: 0
	});
}
