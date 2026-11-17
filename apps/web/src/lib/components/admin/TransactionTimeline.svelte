<script lang="ts">
	import type { TransactionEvent } from '#lib/types/admin-transactions.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	let { title, events }: { title: string; events: TransactionEvent[] } = $props();
	const headingId = $props.id();
</script>

<section
	aria-labelledby={headingId}
	class="mt-6 grid gap-4 border-t border-line pt-6 lg:grid-cols-[minmax(12rem,1fr)_minmax(0,2fr)] lg:gap-8"
>
	<h2 id={headingId} class="text-lg/7 font-semibold">{title}</h2>
	<ol class="min-w-0 rounded-xl bg-page px-4 py-5 sm:px-5">
		{#each events as event (`${event.time}-${event.title}`)}
			<li
				class="relative border-l border-line pr-1 pb-5 pl-4 text-sm before:absolute before:top-2 before:-left-1 before:size-2 before:rounded-full before:bg-primary last:pb-0"
			>
				<p class="font-medium">{event.title}</p>
				<time datetime={event.time} class="mt-1 block text-xs text-ink-muted tabular-nums"
					>{formatWitaDateTime(event.time)} WITA</time
				>
				{#if event.description}<p class="mt-1 text-sm leading-6 text-ink-muted">
						{event.description}
					</p>{/if}
			</li>
		{/each}
	</ol>
</section>
