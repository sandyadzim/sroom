import { PUBLIC_GATEWAY_BASE_URL } from '$app/env/public';
import { resolve } from '$app/paths';
import axios, { isAxiosError } from 'axios';
export { getApiError } from './error.ts';

export const staffApiPath = '/api/v1/staff';
export const catalogAdminPath = `${staffApiPath}/catalog/admin`;

export const staffApi = axios.create({
	baseURL: PUBLIC_GATEWAY_BASE_URL || `http://${globalThis.location?.hostname ?? 'localhost'}:8080`,
	timeout: 12_000,
	withCredentials: true,
	allowAbsoluteUrls: false
});

staffApi.interceptors.response.use(undefined, (cause: unknown) => {
	if (isAxiosError(cause) && cause.response?.status === 401 && typeof window !== 'undefined') {
		const loginPath = resolve('/admin/login');
		if (window.location.pathname !== loginPath) window.location.replace(loginPath);
	}
	throw cause;
});
