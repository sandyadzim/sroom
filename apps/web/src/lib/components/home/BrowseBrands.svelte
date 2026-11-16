<script lang="ts">
	import { resolve } from '$app/paths';
	import Link from '#lib/components/ui/Link.svelte';
	import type { PublicBrand } from '#lib/types/catalog.ts';

	let { brands }: { brands: PublicBrand[] } = $props();
	let failedLogos = $state<Record<string, string | null>>({});
	const logos = $derived(
		brands.filter(({ id, image_url }) => image_url && failedLogos[id] !== image_url).slice(0, 5)
	);
</script>

{#if logos.length}
	<section id="brands" class="bg-panel" aria-label="Car brands">
		<div class="page-container">
			<ul class="flex flex-wrap items-center justify-center border-t border-line py-3 sm:py-5">
				{#each logos as brand (brand.id)}
					<li class="w-1/3 max-w-48 min-w-0 sm:w-1/5">
						<Link
							href={resolve(`/cars?${new URLSearchParams({ brand: brand.id })}`)}
							variant="plain"
							aria-label={`Explore ${brand.name} cars`}
							class="grid min-h-22 place-items-center rounded-lg px-2 py-3 focus-visible:-outline-offset-2 sm:min-h-28 sm:px-5"
						>
							<img
								src={brand.image_url ?? undefined}
								alt=""
								loading="lazy"
								class="h-16 w-full max-w-34 object-contain grayscale group-hover:grayscale-0 group-focus-visible:grayscale-0 group-active:grayscale-0 motion-safe:transition-[filter] motion-safe:duration-300 motion-safe:ease-out sm:h-20"
								onerror={() => (failedLogos[brand.id] = brand.image_url)}
							/>
						</Link>
					</li>
				{/each}
			</ul>
		</div>
	</section>
{/if}
