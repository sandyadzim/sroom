import { GATEWAY_BASE_URL } from '$app/env/private';
import axios from 'axios';

export function createGatewayClient(token?: string) {
	const client = axios.create({
		baseURL: GATEWAY_BASE_URL,
		timeout: 10_000,
		maxRedirects: 0,
		headers: { Accept: 'application/json' }
	});

	if (token) {
		client.interceptors.request.use((config) => {
			config.headers.set('Authorization', `Bearer ${token}`);
			return config;
		});
	}

	return client;
}
