import { staffApi, staffApiPath } from '../staff';
import type { ApiData, LoginCredentials, UserProfile } from '#lib/types/identity.ts';

export async function login(credentials: LoginCredentials) {
	const { data } = await staffApi.post<ApiData<UserProfile>>(`${staffApiPath}/login`, credentials);
	return data.data;
}

export async function logout() {
	await staffApi.post(`${staffApiPath}/logout`);
}
