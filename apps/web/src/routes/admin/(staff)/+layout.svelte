<script lang="ts">
	import AdminShell from '#lib/components/admin/AdminShell.svelte';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
</script>

<AdminShell profile={data.profile}>
	{@render children()}
</AdminShell>
