<script lang="ts">
	const steps = [
		{
			title: 'Find your car.',
			copy: 'Browse the fleet and get to know the car that fits your plans. No dates or account needed to explore.'
		},
		{
			title: 'Make it a date.',
			copy: 'Sign in, then choose your pickup and return dates and times at checkout. Check availability and review your total.'
		},
		{
			title: 'Book your drive.',
			copy: 'Pay in full by Virtual Account within the 15-minute reservation window. Your booking is confirmed after payment is verified.'
		},
		{
			title: 'Take the wheel.',
			copy: 'Once confirmed, keep your booking details and invoice in your account. Pick up in Samarinda and return at the agreed time.'
		}
	];
</script>

<section id="how-it-works" class="bg-panel-muted py-16 sm:py-24" aria-labelledby="how-heading">
	<div class="page-container">
		<div class="grid gap-5 md:grid-cols-2 md:gap-10 lg:gap-20">
			<h2 id="how-heading" class="text-3xl/tight font-semibold tracking-tight sm:text-4xl/tight">
				From browsing<br />to the open road.
			</h2>
			<p class="max-w-lg self-end text-base/7 text-pretty text-ink-muted">
				A car you like. A date in mind. Here’s how your next drive comes together.
			</p>
		</div>
		<ol class="mt-10 grid list-decimal gap-8 pl-5 sm:mt-16 sm:grid-cols-2 sm:gap-10 lg:grid-cols-4">
			{#each steps as step (step.title)}
				<li class="pl-2 marker:text-ink-muted">
					<h3 class="text-xl/7 font-semibold">{step.title}</h3>
					<p class="mt-3 text-base/7 text-pretty text-ink-muted">{step.copy}</p>
				</li>
			{/each}
		</ol>
	</div>
</section>
