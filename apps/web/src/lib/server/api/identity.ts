import type { ApiData, LoginCredentials, LoginSession, UserProfile } from '#lib/types/identity.ts';
import { createGatewayClient } from './gateway';

const identityPath = '/api/v1/identity';
const gateway = createGatewayClient();

export async function login(credentials: LoginCredentials) {
	const response = await gateway.post<ApiData<LoginSession>>(`${identityPath}/login`, credentials);
	return response.data.data;
}

export async function fetchCurrentUser(token: string) {
	const response = await createGatewayClient(token).get<ApiData<UserProfile>>(`${identityPath}/me`);
	return response.data.data;
}

export async function revokeSession(token: string) {
	await createGatewayClient(token).post(`${identityPath}/logout`);
}
