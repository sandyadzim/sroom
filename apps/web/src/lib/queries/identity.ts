import { queryOptions } from '@tanstack/svelte-query';
import { fetchCurrentUser } from '#lib/api/identity/profile.ts';

export const staffProfileQuery = queryOptions({
	queryKey: ['staff-profile'],
	queryFn: ({ signal }) => fetchCurrentUser(signal),
	staleTime: Infinity
});
