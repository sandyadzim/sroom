<script lang="ts">
	import Hero from '#lib/components/home/Hero.svelte';
	import Introduction from '#lib/components/home/Introduction.svelte';
	import FeaturedFleet from '#lib/components/home/FeaturedFleet.svelte';
	import HowItWorks from '#lib/components/home/HowItWorks.svelte';
	import BeforeYouGo from '#lib/components/home/BeforeYouGo.svelte';
	import Faq from '#lib/components/home/Faq.svelte';
</script>

<svelte:head>
	<title>SROOM — Self-drive car rental in Samarinda</title>
	<meta
		name="description"
		content="Find a car for your kind of day with SROOM. Explore self-drive car rentals in Samarinda and plan your next drive."
	/>
</svelte:head>

<Hero />
<Introduction />
<FeaturedFleet />
<HowItWorks />
<BeforeYouGo />
<Faq />
