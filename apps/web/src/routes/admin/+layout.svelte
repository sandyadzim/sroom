<script lang="ts">
	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
</script>

<QueryClientProvider client={queryClient}>
	{@render children()}
</QueryClientProvider>
