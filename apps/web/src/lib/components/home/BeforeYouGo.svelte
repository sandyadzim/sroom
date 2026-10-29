<script lang="ts">
	import Link from '#lib/components/ui/Link.svelte';

	const essentials = [
		{
			title: 'You take the wheel.',
			copy: 'Every rental is self-drive. Pick up and return at one location in Samarinda, with no delivery or driver service.'
		},
		{
			title: 'Every day counts.',
			copy: 'Rental days include both your pickup and return dates. November 10–12 is three rental days, with the full total shown at checkout.'
		},
		{
			title: 'Start with tomorrow.',
			copy: 'Plan your pickup from tomorrow onwards. Choose both dates and times at checkout, all in Samarinda time, WITA.'
		}
	];
</script>

<section class="bg-primary py-16 text-on-primary sm:py-24" aria-labelledby="before-heading">
	<div class="page-container grid gap-10 md:grid-cols-2 lg:gap-20">
		<div>
			<h2 id="before-heading" class="text-4xl/tight font-semibold tracking-tight lg:text-5xl/tight">
				Good plans.<br />Clear terms.
			</h2>
			<p class="mt-6 max-w-sm text-base/7 text-pretty text-on-primary/85">
				A few things to know before you go, so you can focus on the day ahead.
			</p>
			<Link arrow class="mt-6 font-medium focus-visible:outline-on-primary">Read rental terms</Link>
		</div>
		<dl class="grid gap-8">
			{#each essentials as essential (essential.title)}
				<div>
					<dt class="font-display text-xl/7 font-semibold">{essential.title}</dt>
					<dd class="mt-3 max-w-lg text-base/7 text-pretty text-on-primary/85">{essential.copy}</dd>
				</div>
			{/each}
		</dl>
	</div>
</section>
