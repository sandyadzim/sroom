import type { BookingHistoryFilters } from '#lib/types/booking.ts';
import { bookingPageSize } from '#lib/utils/booking-history.ts';
import { queryOptions } from '@tanstack/svelte-query';
import { fetchBooking, fetchCustomerBookings } from '#lib/api/booking.ts';
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

export function customerBookingsQuery(customerId: string, filters: BookingHistoryFilters) {
	const params = {
		page: filters.page,
		limit: bookingPageSize,
		status: filters.status === 'all' ? undefined : filters.status
	};
	return queryOptions({
		queryKey: [...customerRentalKey, customerId, 'bookings', params],
		queryFn: ({ signal }) => fetchCustomerBookings(params, signal),
		enabled: !!customerId,
		retry: false,
		staleTime: 0
	});
}
