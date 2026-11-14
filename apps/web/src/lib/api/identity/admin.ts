import type {
	AdminAccount,
	ApiData,
	CreateAdminInput,
	UpdateAdminInput
} from '#lib/types/identity.ts';
import { staffApi, staffApiPath } from '../staff';

export const adminAccountsEndpoint = `${staffApiPath}/admins`;

export async function fetchAdminAccount(id: string, signal?: AbortSignal) {
	const { data } = await staffApi.get<ApiData<AdminAccount>>(
		`${adminAccountsEndpoint}/${encodeURIComponent(id)}`,
		{ signal }
	);
	return data.data;
}

export async function createAdminAccount(input: CreateAdminInput) {
	const { data } = await staffApi.post<ApiData<AdminAccount>>(adminAccountsEndpoint, input);
	return data.data;
}

export async function updateAdminAccount(id: string, input: UpdateAdminInput) {
	const { data } = await staffApi.patch<ApiData<AdminAccount>>(
		`${adminAccountsEndpoint}/${encodeURIComponent(id)}`,
		input
	);
	return data.data;
}
