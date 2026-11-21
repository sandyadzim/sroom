import { staffApi, staffApiPath } from './staff.ts';
import { fetchTablePage } from './data-table.ts';
import { lookupCustomers } from './identity/customer.ts';
import type {
	StaffBooking,
	StaffBookingDetail,
	StaffPayment,
	StaffPaymentDetail
} from '#lib/types/admin-transactions.ts';
import type { TablePage, TableParams } from '#lib/types/data-table.ts';
import type { ApiData, CustomerLookup } from '#lib/types/identity.ts';
import type { Invoice } from '#lib/types/payment.ts';
import {
	attachCustomerNames,
	bookingPaymentAttemptParams,
	uniqueCustomerIds
} from '#lib/utils/staff-transactions.ts';

export const staffBookingsEndpoint = `${staffApiPath}/bookings`;
export const staffPaymentsEndpoint = `${staffApiPath}/payments`;

function bookingPath(id: string) {
	return `${staffBookingsEndpoint}/${encodeURIComponent(id)}`;
}

function paymentPath(id: string) {
	return `${staffPaymentsEndpoint}/${encodeURIComponent(id)}`;
}

export async function fetchStaffBooking(id: string, signal?: AbortSignal) {
	const { data } = await staffApi.get<ApiData<StaffBookingDetail>>(bookingPath(id), { signal });
	return data.data;
}

export async function fetchStaffBookingPage(params: TableParams, signal?: AbortSignal) {
	const page = await fetchTablePage<StaffBooking>(staffBookingsEndpoint, params, signal);
	return withCustomerNames(page, signal);
}

export async function fetchStaffPaymentPage(params: TableParams, signal?: AbortSignal) {
	const page = await fetchTablePage<StaffPayment>(staffPaymentsEndpoint, params, signal);
	return withCustomerNames(page, signal);
}

export async function fetchStaffPayment(id: string, signal?: AbortSignal) {
	const { data } = await staffApi.get<ApiData<StaffPaymentDetail>>(paymentPath(id), { signal });
	return data.data;
}

export async function fetchStaffInvoice(bookingId: string, signal?: AbortSignal) {
	const { data } = await staffApi.get<ApiData<Invoice>>(`${bookingPath(bookingId)}/invoice`, {
		signal
	});
	return data.data;
}

export async function fetchBookingPaymentAttempts(bookingId: string, signal?: AbortSignal) {
	const { data } = await staffApi.get<TablePage<StaffPayment>>(staffPaymentsEndpoint, {
		params: bookingPaymentAttemptParams(bookingId),
		signal
	});
	return data;
}

async function withCustomerNames<T extends { customer_id: string }>(
	page: TablePage<T>,
	signal?: AbortSignal
): Promise<TablePage<T & { customer_name: string | null }>> {
	const ids = uniqueCustomerIds(page.data);
	let customers: CustomerLookup[] = [];
	if (ids.length) {
		try {
			customers = await lookupCustomers(ids, signal);
		} catch (cause) {
			if (signal?.aborted) throw cause;
		}
	}
	return { ...page, data: attachCustomerNames(page.data, customers) };
}
