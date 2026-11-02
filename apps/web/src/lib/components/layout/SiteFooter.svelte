<script lang="ts">
	import { ArrowUp } from '@lucide/svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import Wordmark from './Wordmark.svelte';
	import { resolve } from '$app/paths';

	const navigation = [
		{ label: 'Explore Cars', href: resolve('/cars') },
		{ label: 'Rental Terms', href: resolve('/rental-terms') },
		{ label: 'My Bookings', href: '#' }
	];
</script>

<footer id="footer" class="mt-16 px-2 sm:mt-24" aria-label="SROOM footer">
	<div class="overflow-hidden rounded-t-2xl bg-primary text-on-primary sm:rounded-t-3xl">
		<div class="mx-auto max-w-7xl px-6 pt-10 sm:px-8 sm:pt-16 lg:px-10">
			<div class="flex flex-col items-start justify-between gap-6 lg:flex-row lg:gap-10">
				<nav aria-label="Footer navigation">
					<ul class="flex flex-col gap-1 sm:flex-row sm:flex-wrap sm:gap-x-6 sm:gap-y-2">
						{#each navigation as link (link.label)}
							<li>
								<Link href={link.href} variant="footer" arrow>{link.label}</Link>
							</li>
						{/each}
					</ul>
				</nav>
				<div class="space-y-2 text-sm leading-6 text-on-primary/80">
					<p>Self-drive in Samarinda · WITA.</p>
				</div>
			</div>
			<div
				class="mt-8 flex items-center justify-between gap-4 border-t border-on-primary/20 pt-6 text-sm leading-6 text-on-primary/80 sm:mt-10"
			>
				<p>© 2026 SROOM.</p>
				<Link class="hover:text-on-primary">
					Back to top <ArrowUp size={16} strokeWidth={1.5} aria-hidden="true" />
				</Link>
			</div>
			<Wordmark size="display" class="mt-8" />
		</div>
	</div>
</footer>
