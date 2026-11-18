<script lang="ts">
	import { formatWitaDate } from '#lib/utils/date.ts';

	let {
		title,
		data,
		format = (value: number) => String(value),
		unit = 1
	}: {
		title: string;
		data: { day: string; value: number }[];
		format?: (value: number) => string;
		unit?: number;
	} = $props();
	const headingId = $props.id();
	const maximum = $derived(Math.ceil(Math.max(0, ...data.map(({ value }) => value)) / unit) * unit);
</script>

<section
	aria-labelledby={headingId}
	class="min-w-0 rounded-xl border border-line bg-panel p-4 sm:p-5"
>
	<h2 id={headingId} class="text-lg font-semibold tracking-tight">{title}</h2>
	{#if maximum === 0}
		<p class="mt-5 rounded-lg bg-panel-muted px-4 py-5 text-sm text-ink-muted">
			No activity to chart for this period.
		</p>
	{:else}
		<div aria-hidden="true" class="mt-5 grid grid-cols-[auto_minmax(0,1fr)] gap-2 sm:gap-3">
			<div
				class="flex h-34 flex-col justify-between text-right text-[11px] leading-none text-ink-muted tabular-nums"
			>
				<span>{format(maximum)}</span><span>{format(0)}</span>
			</div>
			<div class="min-w-0">
				<div
					class="relative flex h-34 items-end gap-1 border-y border-line px-0.5 sm:gap-2 sm:px-2"
				>
					<div
						class="pointer-events-none absolute inset-x-0 top-1/2 border-t border-dashed border-line"
					></div>
					{#each data as point (point.day)}
						<div
							class="relative h-full min-w-0 flex-1"
							title={`${formatWitaDate(point.day)} · ${format(point.value)}`}
						>
							<span
								class="absolute inset-x-0 bottom-0 mx-auto block max-w-12 rounded-t-sm bg-primary"
								style:height={`${(100 * point.value) / maximum}%`}
							></span>
						</div>
					{/each}
				</div>
				<div
					class="mt-2 flex gap-1 px-0.5 text-center text-[11px] text-ink-muted tabular-nums sm:gap-2 sm:px-2"
				>
					{#each data as point (point.day)}
						<span class="min-w-0 flex-1">{point.day.slice(-2)}</span>
					{/each}
				</div>
			</div>
		</div>
		<p class="mt-2 text-xs text-ink-muted">Day of month · WITA</p>
		<details class="mt-3 border-t border-line pt-2 text-sm">
			<summary class="cursor-pointer rounded-sm py-2 font-medium text-primary"
				>View daily values</summary
			>
			<dl class="space-y-1 pb-2">
				{#each data as point (point.day)}
					<div class="flex flex-wrap justify-between gap-x-3">
						<dt class="text-ink-muted">{formatWitaDate(point.day)}</dt>
						<dd class="font-medium tabular-nums">{format(point.value)}</dd>
					</div>
				{/each}
			</dl>
		</details>
	{/if}
</section>
