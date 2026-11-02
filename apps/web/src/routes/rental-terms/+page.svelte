<script lang="ts">
	import Link from '#lib/components/ui/Link.svelte';

	const sections = [
		{
			id: 'before-you-book',
			title: 'Before you book',
			paragraphs: [
				'SROOM offers self-drive car rentals in Samarinda. You can browse cars without an account or selected dates. Sign in or create an account when you’re ready to book; guest checkout is not available.',
				'Bring your identity card (KTP) and a valid driving licence (SIM) for an in-person check at pickup. You do not need to upload these documents online.'
			]
		},
		{
			id: 'dates-and-pricing',
			title: 'Dates and pricing',
			paragraphs: [
				'Choose your pickup and return dates and times at checkout, where availability is checked for your selected car. The earliest pickup date is tomorrow in Samarinda. All rental times use WITA (UTC+08:00).',
				'Both the pickup date and the return date count as rental days. For example, November 10 through November 12 is three rental days, regardless of the agreed pickup and return times. Your total is the daily rate multiplied by the number of rental days, shown in Indonesian Rupiah (IDR) before payment.'
			]
		},
		{
			id: 'reservations-and-payment',
			title: 'Reservations and payment',
			paragraphs: [
				'Pay the full rental amount upfront by Virtual Account. No deposit is required. Once a reservation is created, you have 15 minutes to complete payment. Follow the deadline and instructions shown with your payment details.',
				'Your booking is confirmed only after payment is verified and the booking status is confirmed. Starting a payment or returning from a payment page does not confirm a booking. Check your booking status before making pickup plans.'
			]
		},
		{
			id: 'pickup-and-return',
			title: 'Pickup and return',
			paragraphs: [
				'Pick up and return your car at the same location in Samarinda, on the dates and at the times agreed in your booking. Rentals are self-drive only; delivery and driver services are not included.',
				'Review your car, schedule, and total before paying. The pickup and return times you agree to are included in the rental price; a longer agreed time within those dates does not add an hourly surcharge at checkout.'
			]
		},
		{
			id: 'late-returns',
			title: 'Late returns',
			paragraphs: [
				'Overtime applies only if the car is actually returned after the agreed return time. It is not calculated from your selected pickup and return hours or added to the booking total in advance.',
				'Any overtime charge is reviewed and handled manually, outside the online payment flow. The website does not automatically calculate or collect late-return charges.'
			]
		},
		{
			id: 'changes-and-refunds',
			title: 'Changes and refunds',
			paragraphs: [
				'Bookings cannot be cancelled or rescheduled through your account. Check your selected car, dates, times, and full total carefully before you confirm and pay.',
				'Any exceptional refund requires a manual review and is processed separately. A payment or booking status does not, by itself, mean a refund has been approved or completed.'
			]
		},
		{
			id: 'booking-records',
			title: 'Your booking records',
			paragraphs: [
				'Your account holds your booking details, agreed schedule, and payment status. Your invoice becomes available after your booking is confirmed. Booking confirmation and invoice emails are sent after confirmation.',
				'Keep your booking details available for pickup and return. Review these terms alongside your booking summary before confirming a reservation.'
			]
		}
	];
</script>

<svelte:head>
	<title>Rental Terms — SROOM</title>
	<meta
		name="description"
		content="Read SROOM’s rental terms for self-drive car hire in Samarinda, including booking, daily pricing, payment, pickup, return, and changes."
	/>
</svelte:head>

<div class="page-container py-12 sm:py-16 lg:py-20">
	<div class="mx-auto max-w-5xl">
		<div class="max-w-2xl">
			<h1 class="text-4xl/tight font-semibold tracking-tight sm:text-5xl/tight">Rental terms.</h1>
			<p class="mt-5 max-w-xl text-base/7 text-pretty text-ink-muted sm:text-lg/8">
				Clear plans make for a better drive. Here’s what to know about booking, picking up, and
				returning your car with SROOM.
			</p>
		</div>

		<div
			class="mt-10 grid items-start gap-10 sm:mt-16 lg:grid-cols-[16rem_minmax(0,1fr)] lg:gap-20"
		>
			<nav
				aria-labelledby="terms-contents"
				class="rounded-panel bg-panel-muted p-6 lg:sticky lg:top-32"
			>
				<h2 id="terms-contents" class="text-lg/7 font-semibold">On this page</h2>
				<ul class="mt-4 grid grid-cols-2 gap-x-4 sm:gap-x-6 lg:grid-cols-1">
					{#each sections as section (section.id)}
						<li>
							<Link
								href={`#${section.id}`}
								variant="plain"
								class="flex min-h-11 items-center py-2 text-sm hover:underline"
							>
								{section.title}
							</Link>
						</li>
					{/each}
				</ul>
			</nav>

			<div class="min-w-0 space-y-12 sm:space-y-16">
				{#each sections as section (section.id)}
					<section id={section.id} aria-labelledby={`${section.id}-heading`} tabindex="-1">
						<h2 id={`${section.id}-heading`} class="text-2xl/8 font-semibold tracking-tight">
							{section.title}
						</h2>
						<div class="mt-4 space-y-4 text-base/7">
							{#each section.paragraphs as paragraph (paragraph)}
								<p>{paragraph}</p>
							{/each}
						</div>
					</section>
				{/each}
			</div>
		</div>
	</div>
</div>
