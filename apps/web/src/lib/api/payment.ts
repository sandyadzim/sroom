import { customerApi, customerPath } from './customer.ts';
import type { ApiData } from '#lib/types/identity.ts';
import type { CustomerPayment, Invoice, VirtualAccount } from '#lib/types/payment.ts';

function bookingPath(id: string) {
	return `${customerPath}/bookings/${encodeURIComponent(id)}`;
}
export async function fetchCustomerPayment(id: string, signal?: AbortSignal) {
	const { data } = await customerApi.get<ApiData<CustomerPayment>>(`${bookingPath(id)}/payment`, {
		signal
	});
	return data.data;
}
export async function createVirtualAccount(id: string, key: string) {
	const { data } = await customerApi.post<ApiData<VirtualAccount>>(
		`${bookingPath(id)}/payments/va`,
		{},
		{ headers: { 'Idempotency-Key': key } }
	);
	return data.data;
}
export async function refreshCustomerPayment(id: string) {
	const { data } = await customerApi.post<ApiData<CustomerPayment>>(
		`${bookingPath(id)}/payment/refresh`,
		{}
	);
	return data.data;
}
export async function fetchCustomerInvoice(id: string, signal?: AbortSignal) {
	const { data } = await customerApi.get<ApiData<Invoice>>(`${bookingPath(id)}/invoice`, {
		signal
	});
	return data.data;
}
