import { PUBLIC_GATEWAY_BASE_URL } from '$app/env/public';
import axios, { isAxiosError } from 'axios';
import type {
	ApiData,
	CustomerSessionProfile,
	LoginCredentials,
	RegisteredCustomer,
	RegisterCustomerInput
} from '#lib/types/identity.ts';

const customerPath = '/api/v1/customer';
const api = axios.create({
	baseURL: PUBLIC_GATEWAY_BASE_URL || `http://${globalThis.location?.hostname ?? 'localhost'}:8080`,
	timeout: 12_000,
	allowAbsoluteUrls: false
});

export async function registerCustomer(input: RegisterCustomerInput) {
	const { data } = await api.post<ApiData<RegisteredCustomer>>('/api/v1/identity/register', input);
	return data.data;
}

export async function loginCustomer(input: LoginCredentials) {
	const { data } = await api.post<ApiData<CustomerSessionProfile>>(`${customerPath}/login`, input, {
		withCredentials: true
	});
	return data.data;
}

export async function fetchCustomerProfile(signal?: AbortSignal) {
	try {
		const { data } = await api.get<ApiData<CustomerSessionProfile>>(`${customerPath}/me`, {
			withCredentials: true,
			signal
		});
		return data.data;
	} catch (cause) {
		if (isAxiosError(cause) && cause.response?.status === 401) return null;
		throw cause;
	}
}

export async function logoutCustomer() {
	await api.post(`${customerPath}/logout`, undefined, { withCredentials: true });
}
