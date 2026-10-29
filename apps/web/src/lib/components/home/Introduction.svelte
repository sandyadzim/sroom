<script lang="ts">
	import { ArrowUpRight } from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
</script>

<section class="bg-panel-muted py-16 sm:py-24" aria-labelledby="introduction-heading">
	<div class="page-container grid items-center gap-10 md:grid-cols-2 lg:gap-20">
		<div>
			<h2
				id="introduction-heading"
				class="text-4xl/tight font-semibold tracking-tight text-primary lg:text-5xl/tight xl:text-6xl/tight"
			>
				Take the<br />long way home.
			</h2>
			<p class="mt-6 max-w-lg text-lg/8 text-pretty">
				Some days are about getting there. Others are about everything along the way.
			</p>
			<p class="mt-4 max-w-lg text-base/7 text-pretty text-ink-muted">
				SROOM is a self-drive car rental experience in Samarinda. Find a car for the day you have in
				mind, from a full calendar to a wide-open weekend. You bring the plans. You take the wheel.
			</p>
			<Button href="#" class="mt-8">
				Explore Cars <ArrowUpRight size={18} strokeWidth={1.5} aria-hidden="true" />
			</Button>
		</div>
		<figure class="min-w-0 md:order-first">
			<img
				src="/images/introduction/journey.webp"
				alt="Illustration of a white SUV on a winding road through a tropical forest"
				width="1600"
				height="1066"
				loading="lazy"
				class="aspect-4/3 w-full rounded-panel object-cover object-right sm:aspect-3/2 md:aspect-square"
			/>
		</figure>
	</div>
</section>
