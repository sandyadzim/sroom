import { queryOptions } from '@tanstack/svelte-query';
import { fetchAdminAccount } from '#lib/api/identity/admin.ts';
import { fetchCurrentUser } from '#lib/api/identity/profile.ts';

export const staffProfileQuery = queryOptions({
	queryKey: ['staff-profile'],
	queryFn: ({ signal }) => fetchCurrentUser(signal),
	staleTime: Infinity
});

export function adminAccountQuery(id: string) {
	return queryOptions({
		queryKey: ['admin-account', id],
		queryFn: ({ signal }) => fetchAdminAccount(id, signal)
	});
}
