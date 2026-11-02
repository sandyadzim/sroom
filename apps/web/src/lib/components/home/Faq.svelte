<script lang="ts">
	import AccordionItem from '#lib/components/ui/AccordionItem.svelte';

	const questions = [
		{
			question: 'Do I need an account?',
			answer:
				'You can explore cars and their details without one. Sign in or create an account when you’re ready to book. Guest checkout isn’t available.'
		},
		{
			question: 'When can I pick up a car?',
			answer:
				'The earliest pickup date is tomorrow in Samarinda. Choose your pickup and return dates and times at checkout, where availability is checked. All rental times are shown in WITA (UTC+08:00).'
		},
		{
			question: 'How are rental days counted?',
			answer:
				'Both the pickup date and return date count. For example, November 10 through November 12 is three rental days, regardless of the agreed pickup and return times. Your final total is calculated at checkout.'
		},
		{
			question: 'Is a deposit required?',
			answer:
				'No deposit is required. The full rental amount is paid upfront by Virtual Account, with your total shown before payment.'
		},
		{
			question: 'When is my booking confirmed?',
			answer:
				'Your reservation has a 15-minute payment window. A booking is confirmed only after payment is verified and the booking status is confirmed. Starting a payment or returning to the website does not confirm your booking.'
		},
		{
			question: 'Can I cancel or change my dates?',
			answer:
				'Bookings cannot be cancelled or rescheduled through your account. Review your car, dates, times, and total carefully before paying. Any exceptional refund is reviewed and processed manually.'
		}
	];
</script>

<section id="faq" class="bg-panel py-16 sm:py-24" aria-labelledby="faq-heading">
	<div class="page-container grid items-start gap-10 md:grid-cols-[1fr_1.5fr] lg:gap-20">
		<div>
			<h2 id="faq-heading" class="text-3xl/tight font-semibold tracking-tight sm:text-4xl/tight">
				A few good<br />questions.
			</h2>
			<p class="mt-5 max-w-xs text-base/7 text-pretty text-ink-muted">
				The details worth knowing before you book your next drive.
			</p>
		</div>
		<div class="grid min-w-0 gap-3">
			{#each questions as item (item.question)}
				<AccordionItem question={item.question}>
					<p>{item.answer}</p>
				</AccordionItem>
			{/each}
		</div>
	</div>
</section>
