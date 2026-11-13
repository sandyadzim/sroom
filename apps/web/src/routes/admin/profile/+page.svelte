<script lang="ts">
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Staff profile — SROOM</title>
</svelte:head>

<pre class="overflow-x-auto p-6 text-sm">{JSON.stringify(data.profile, null, 2)}</pre>
