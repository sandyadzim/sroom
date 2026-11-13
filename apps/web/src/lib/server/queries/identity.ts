import { QueryClient } from '@tanstack/svelte-query';
import { fetchCurrentUser } from '#lib/server/api/identity.ts';

export function getCurrentUser(token: string) {
	// Keep the profile query cache scoped to this server request.
	return new QueryClient().query({
		queryKey: ['me'],
		queryFn: () => fetchCurrentUser(token)
	});
}
