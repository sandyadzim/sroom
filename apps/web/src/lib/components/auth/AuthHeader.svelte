<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';
	import HeaderRow from '#lib/components/layout/HeaderRow.svelte';
	import Wordmark from '#lib/components/layout/Wordmark.svelte';
	import Link from '#lib/components/ui/Link.svelte';
</script>

<header class="bg-panel">
	<HeaderRow>
		<Wordmark href={resolve('/')} />
		<Link href={resolve('/')} class="gap-2 text-sm">
			<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" /> Back to home
		</Link>
	</HeaderRow>
</header>
