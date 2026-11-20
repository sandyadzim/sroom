import { PUBLIC_GATEWAY_BASE_URL } from '$app/env/public';
import axios from 'axios';

export const customerPath = '/api/v1/customer';
export const customerApi = axios.create({
	baseURL: PUBLIC_GATEWAY_BASE_URL || `http://${globalThis.location?.hostname ?? 'localhost'}:8080`,
	timeout: 12_000,
	withCredentials: true,
	allowAbsoluteUrls: false
});
