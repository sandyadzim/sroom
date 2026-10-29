<script lang="ts">
	import { Menu, X, ArrowUpRight } from '@lucide/svelte';
	import Wordmark from './Wordmark.svelte';
	import HeaderRow from './HeaderRow.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { afterNavigate } from '$app/navigation';
	import { onMount } from 'svelte';

	const links = ['Explore Cars', 'How It Works', 'Rental Terms'];
	let menu: HTMLDialogElement;
	let menuOpen = $state(false);
	let closing = $state(false);
	const reducedMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches;

	function openMenu() {
		if (menu.open) return;
		closing = false;
		menu.showModal();
		menuOpen = true;
	}

	function closeMenu() {
		if (!menu?.open || closing) return;
		// No transition will finish if motion is disabled or the menu has not appeared yet.
		if (reducedMotion() || getComputedStyle(menu).opacity === '0') {
			menu.close();
			return;
		}
		closing = true;
	}

	function finishClose(event: TransitionEvent) {
		if (event.target !== menu || event.propertyName !== 'opacity' || !closing) return;
		if (event.type === 'transitionend' || reducedMotion()) menu.close();
	}

	afterNavigate(closeMenu);

	$effect(() => {
		if (!menuOpen) return;
		const previousOverflow = document.body.style.overflow;
		document.body.style.overflow = 'hidden';
		return () => {
			document.body.style.overflow = previousOverflow;
		};
	});

	onMount(() => {
		const desktop = window.matchMedia('(min-width: 64rem)');
		const handleResize = () => {
			if (desktop.matches) closeMenu();
		};
		desktop.addEventListener('change', handleResize);
		return () => desktop.removeEventListener('change', handleResize);
	});
</script>

{#snippet navigation(expanded = false)}
	<nav
		aria-label={expanded ? 'Mobile main navigation' : 'Main navigation'}
		class={expanded ? 'flex flex-col gap-4' : 'flex items-center gap-4 text-sm xl:gap-6'}
	>
		{#each links as label (label)}
			<Link
				variant="nav"
				arrow={expanded}
				onclick={closeMenu}
				class={expanded
					? 'min-h-12 justify-between font-display text-xl font-semibold tracking-tight'
					: undefined}
			>
				{label}
			</Link>
		{/each}
	</nav>
{/snippet}

{#snippet accountActions(expanded = false)}
	<div
		class={expanded
			? 'flex flex-col items-stretch gap-4 text-sm'
			: 'flex items-center gap-4 text-sm xl:gap-6'}
	>
		<Link onclick={closeMenu}>Sign In</Link>
		<Button
			href="#"
			onclick={closeMenu}
			variant={expanded ? 'inverse' : 'primary'}
			class="whitespace-nowrap"
		>
			Create Account <ArrowUpRight size={18} strokeWidth={1.5} aria-hidden="true" />
		</Button>
	</div>
{/snippet}

<div class="sticky top-0 z-40">
	<header class="bg-panel" data-header="balanced-airy">
		<HeaderRow>
			<Wordmark />
			<div class="hidden lg:block">{@render navigation()}</div>
			<div class="hidden lg:block">{@render accountActions()}</div>
			<div class="flex items-center gap-3 lg:hidden">
				<Button
					variant="ghost"
					size="icon"
					aria-label="Open navigation"
					aria-haspopup="dialog"
					aria-controls="mobile-navigation"
					aria-expanded={menuOpen}
					onclick={openMenu}
				>
					<Menu size={22} strokeWidth={1.5} aria-hidden="true" />
				</Button>
			</div>
		</HeaderRow>
	</header>
	<dialog
		bind:this={menu}
		id="mobile-navigation"
		data-closing={closing || undefined}
		class="fixed inset-0 m-0 h-dvh max-h-none w-full max-w-none overflow-y-auto overscroll-contain border-0 bg-primary p-0 text-on-primary backdrop:bg-transparent"
		aria-label="Site navigation"
		ontransitionend={finishClose}
		ontransitioncancel={finishClose}
		oncancel={(event) => {
			event.preventDefault();
			closeMenu();
		}}
		onclose={() => {
			menuOpen = false;
			closing = false;
		}}
	>
		<div class="flex min-h-full flex-col pb-safe">
			<HeaderRow>
				<Wordmark inverse onclick={closeMenu} />
				<Button
					variant="ghost-inverse"
					size="icon"
					aria-label="Close navigation"
					onclick={closeMenu}
				>
					<X size={22} strokeWidth={1.5} aria-hidden="true" />
				</Button>
			</HeaderRow>
			<div class="page-container flex flex-1 flex-col pb-8 sm:pb-10">
				<div class="py-10">{@render navigation(true)}</div>
				<div>{@render accountActions(true)}</div>
			</div>
		</div>
	</dialog>
</div>
