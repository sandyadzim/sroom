import { staffApi, staffApiPath } from '#lib/api/staff.ts';
import type { ApiData, CustomerAccount, CustomerLookup } from '#lib/types/identity.ts';

export const customersEndpoint = `${staffApiPath}/customers`;

export async function lookupCustomers(ids: string[], signal?: AbortSignal) {
	const { data } = await staffApi.post<{ data: CustomerLookup[] }>(
		`${customersEndpoint}/lookup`,
		{ customer_ids: ids },
		{ signal }
	);
	return data.data;
}

export async function fetchCustomer(id: string, signal: AbortSignal) {
	const { data } = await staffApi.get<ApiData<CustomerAccount>>(
		`${customersEndpoint}/${encodeURIComponent(id)}`,
		{ signal }
	);
	return data.data;
}

export async function updateCustomerStatus(id: string, status: CustomerAccount['status']) {
	const { data } = await staffApi.patch<ApiData<CustomerAccount>>(
		`${customersEndpoint}/${encodeURIComponent(id)}/status`,
		{ status }
	);
	return data.data;
}
