import { queryOptions } from '@tanstack/svelte-query';
import { fetchCustomerProfile } from '#lib/api/identity/customer-auth.ts';

export const customerProfileQuery = queryOptions({
	queryKey: ['customer-profile'],
	queryFn: ({ signal }) => fetchCustomerProfile(signal),
	enabled: typeof window !== 'undefined',
	staleTime: 30_000,
	retry: false
});
