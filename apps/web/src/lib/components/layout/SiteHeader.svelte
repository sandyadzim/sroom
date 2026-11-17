<script lang="ts">
	import {
		ArrowUpRight,
		CalendarDays,
		ChevronDown,
		LogOut,
		Menu,
		UserRound,
		X
	} from '@lucide/svelte';
	import { Popover } from 'bits-ui';
	import Wordmark from './Wordmark.svelte';
	import HeaderRow from './HeaderRow.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { afterNavigate } from '$app/navigation';
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { logoutCustomer } from '#lib/api/identity/customer-auth.ts';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { useToast } from '#lib/ui/toast.ts';

	const signOutError = 'Could not sign out. Please try again.';
	const profile = createQuery(() => customerProfileQuery);
	const queryClient = useQueryClient();
	const toast = useToast();
	const signOut = createMutation(() => ({
		mutationFn: logoutCustomer,
		retry: false,
		onSuccess: async () => {
			await queryClient.cancelQueries({ queryKey: customerProfileQuery.queryKey });
			queryClient.setQueryData(customerProfileQuery.queryKey, null);
			closeMenus();
		},
		onError: () => {
			if (!menuOpen || closing) toast({ kind: 'error', message: signOutError });
		}
	}));

	const links = [
		{ label: 'Explore Cars', href: resolve('/cars') },
		{ label: 'How It Works', href: `${resolve('/')}#how-it-works` },
		{ label: 'Rental Terms', href: resolve('/rental-terms') }
	];
	let menu: HTMLDialogElement;
	let menuOpen = $state(false);
	let accountOpen = $state(false);
	const profileHref = $derived(resolve('/account/profile'));
	let closing = $state(false);
	const reducedMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches;

	function openMenu() {
		if (menu.open) return;
		if (signOut.isError) signOut.reset();
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

	function closeMenus() {
		accountOpen = false;
		closeMenu();
	}

	afterNavigate(closeMenus);

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
		{#each links as link (link.label)}
			<Link
				href={link.href}
				aria-current={page.url.pathname === link.href ? 'page' : undefined}
				variant="nav"
				arrow={expanded}
				onclick={closeMenu}
				class={expanded
					? 'min-h-12 justify-between font-display text-xl font-semibold tracking-tight'
					: undefined}
			>
				{link.label}
			</Link>
		{/each}
	</nav>
{/snippet}

{#snippet signedInActions(expanded = false)}
	<nav aria-label="Customer account" class={expanded ? 'border-t border-on-primary/20 pt-3' : ''}>
		<Link
			href={profileHref}
			variant="plain"
			aria-current={page.url.pathname === profileHref ? 'page' : undefined}
			onclick={closeMenus}
			class={[
				'flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm font-medium transition-colors motion-reduce:transition-none',
				expanded
					? 'hover:bg-on-primary/10 aria-[current=page]:bg-on-primary/10'
					: 'hover:bg-panel-muted aria-[current=page]:bg-panel-muted'
			]}
		>
			<UserRound size={17} aria-hidden="true" />
			Profile
		</Link>
		<span
			aria-disabled="true"
			class={[
				'flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm',
				expanded ? 'text-on-primary/75' : 'text-ink-muted'
			]}
		>
			<CalendarDays size={17} aria-hidden="true" />
			My Bookings
			<span class="ml-auto text-xs">Coming soon</span>
		</span>
		<div
			class={expanded
				? 'mt-2 border-t border-on-primary/20 pt-2'
				: 'mt-1 border-t border-line pt-1'}
		>
			<Button
				variant={expanded ? 'ghost-inverse' : 'ghost'}
				size="custom"
				class={['w-full justify-start rounded-lg px-3', !expanded && 'text-negative']}
				disabled={signOut.isPending}
				onclick={() => signOut.mutate()}
			>
				<LogOut size={17} aria-hidden="true" />
				{signOut.isPending ? 'Signing out…' : 'Sign Out'}
			</Button>
			{#if expanded && signOut.isError}
				<p role="alert" class="px-3 pt-1 text-sm text-on-primary">{signOutError}</p>
			{/if}
		</div>
	</nav>
{/snippet}

{#snippet accountActions(expanded = false)}
	<div
		class={expanded
			? 'flex flex-col items-stretch gap-4 text-sm'
			: 'flex items-center gap-4 text-sm xl:gap-6'}
	>
		{#if profile.data}
			{@const name = profile.data.full_name.trim() || profile.data.email}
			{#if expanded}
				<p class="truncate px-3 font-medium">{name}</p>
				{@render signedInActions(true)}
			{:else}
				<Popover.Root bind:open={accountOpen}>
					<Popover.Trigger
						class="group inline-flex min-h-11 items-center gap-2.5 rounded-full bg-panel-muted px-4 text-sm font-semibold text-primary transition-colors hover:bg-line motion-reduce:transition-none"
					>
						<span class="max-w-36 truncate xl:max-w-44">{name}</span>
						<ChevronDown
							size={16}
							aria-hidden="true"
							class="shrink-0 transition-transform group-data-[state=open]:rotate-180 motion-reduce:transition-none"
						/>
					</Popover.Trigger>
					<Popover.Portal>
						<Popover.Content
							aria-label="Account menu"
							align="end"
							sideOffset={8}
							class="z-50 w-64 rounded-2xl bg-panel p-1.5 shadow-lg"
						>
							{@render signedInActions()}
						</Popover.Content>
					</Popover.Portal>
				</Popover.Root>
			{/if}
		{:else}
			<Link href={resolve('/(auth)/login')} onclick={closeMenu}>Sign In</Link>
			<Button
				href={resolve('/(auth)/register')}
				onclick={closeMenu}
				variant={expanded ? 'inverse' : 'primary'}
				class="whitespace-nowrap"
			>
				Create Account <ArrowUpRight size={18} strokeWidth={1.5} aria-hidden="true" />
			</Button>
		{/if}
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
		class="fixed inset-0 m-0 h-dvh max-h-none w-full max-w-none overflow-y-auto overscroll-contain border-0 bg-primary p-0 text-on-primary backdrop:bg-transparent [&_:focus-visible]:outline-on-primary"
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
