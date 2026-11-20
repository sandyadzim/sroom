import { isAxiosError } from 'axios';
import { customerApi as api, customerPath } from '../customer.ts';
import type {
	ApiData,
	ChangePasswordInput,
	CustomerSessionProfile,
	LoginCredentials,
	RegisteredCustomer,
	RegisterCustomerInput
} from '#lib/types/identity.ts';

export async function registerCustomer(input: RegisterCustomerInput) {
	// Registration uses the public Identity route, which does not accept cookie credentials.
	const { data } = await api.post<ApiData<RegisteredCustomer>>('/api/v1/identity/register', input, {
		withCredentials: false
	});
	return data.data;
}

export async function loginCustomer(input: LoginCredentials) {
	const { data } = await api.post<ApiData<CustomerSessionProfile>>(`${customerPath}/login`, input);
	return data.data;
}

export async function fetchCustomerProfile(signal?: AbortSignal) {
	try {
		const { data } = await api.get<ApiData<CustomerSessionProfile>>(`${customerPath}/me`, {
			signal
		});
		return data.data;
	} catch (cause) {
		if (isAxiosError(cause) && cause.response?.status === 401) return null;
		throw cause;
	}
}

export async function logoutCustomer() {
	await api.post(`${customerPath}/logout`, undefined);
}

export async function changeCustomerPassword(input: ChangePasswordInput) {
	await api.post(`${customerPath}/change-password`, input);
}
