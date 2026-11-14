import { staffApi, staffApiPath } from '../staff';
import type { ApiData, ChangePasswordInput, UserProfile } from '#lib/types/identity.ts';

export async function fetchCurrentUser(signal?: AbortSignal) {
	const { data } = await staffApi.get<ApiData<UserProfile>>(`${staffApiPath}/me`, { signal });
	return data.data;
}

export async function changePassword(input: ChangePasswordInput) {
	await staffApi.post(`${staffApiPath}/change-password`, input);
}
