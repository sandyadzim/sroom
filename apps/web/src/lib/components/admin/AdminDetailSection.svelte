<script lang="ts">
	import type { Snippet } from 'svelte';

	let { title, children }: { title: string; children: Snippet } = $props();
	const headingId = $props.id();
</script>

<section
	aria-labelledby={headingId}
	class="mt-6 grid gap-4 border-t border-line pt-6 lg:grid-cols-[minmax(12rem,1fr)_minmax(0,2fr)] lg:gap-8"
>
	<h2 id={headingId} class="text-lg/7 font-semibold">{title}</h2>
	<dl class="min-w-0 divide-y divide-line rounded-xl bg-page px-4 sm:px-5">
		{@render children()}
	</dl>
</section>
