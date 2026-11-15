<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import {
		ArrowUpRight,
		ChartColumn,
		CalendarDays,
		CarFront,
		CreditCard,
		LogOut,
		PanelsTopLeft,
		ExternalLink,
		Tag,
		ContactRound,
		UsersRound,
		UserRound,
		Warehouse
	} from '@lucide/svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';

	let {
		isOwner,
		profileHref,
		websiteHref,
		collapsed = false,
		onNavigate,
		onSignOut,
		signingOut
	}: {
		isOwner: boolean;
		profileHref: string;
		websiteHref: string;
		collapsed?: boolean;
		onNavigate?: () => void;
		onSignOut: () => void;
		signingOut: boolean;
	} = $props();

	const operations = [
		{ label: 'Car models', icon: CarFront },
		{ label: 'Physical units', icon: Warehouse },
		{ label: 'Bookings', icon: CalendarDays },
		{ label: 'Payments', icon: CreditCard },
		{ label: 'Reports', icon: ChartColumn }
	];
	const catalogLinks = [
		{ label: 'Brands', icon: Tag, route: '/admin/(staff)/brands' },
		{ label: 'Banners', icon: PanelsTopLeft, route: '/admin/(staff)/banners' }
	] as const;
	const customersHref = $derived(resolve('/admin/(staff)/customers'));
	const inCustomers = $derived(page.route.id?.startsWith('/admin/(staff)/customers'));
	const adminsHref = $derived(resolve('admin/admins'));
	const adminsPath = $derived(new URL(adminsHref, page.url.href).pathname);
	const onAdmins = $derived(page.url.pathname === adminsPath);
	const inAdmins = $derived(onAdmins || page.url.pathname.startsWith(`${adminsPath}/`));
	const profilePath = $derived(new URL(profileHref, page.url.href).pathname);
	const onProfile = $derived(page.url.pathname === profilePath);
	const inProfile = $derived(onProfile || page.url.pathname.startsWith(`${profilePath}/`));
	const row =
		'flex min-h-11 items-center gap-2 overflow-hidden rounded-lg px-3 text-sm focus-visible:-outline-offset-2';
	const labelClass = $derived([
		'whitespace-nowrap motion-safe:transition-opacity motion-safe:duration-150',
		collapsed && 'opacity-0'
	]);
</script>

<nav class="flex flex-1 flex-col lg:min-h-0" aria-label="Staff navigation">
	<div class="space-y-6 lg:overflow-y-auto">
		<div>
			<p class={[labelClass, 'overflow-hidden px-3 pb-2 text-xs font-medium text-ink-muted']}>
				Operations
			</p>
			<ul class="space-y-1">
				{#each catalogLinks as { label, icon: Icon, route } (route)}
					{@const onPage = page.route.id === route}
					{@const inModule = onPage || page.route.id?.startsWith(`${route}/`)}
					<li>
						<Link
							href={resolve(route)}
							variant="plain"
							aria-current={onPage ? 'page' : inModule ? 'location' : undefined}
							title={collapsed ? label : undefined}
							onclick={onNavigate}
							class={[
								row,
								inModule
									? 'bg-panel-muted font-semibold text-ink'
									: 'text-ink-muted hover:bg-panel-muted hover:text-ink'
							]}
						>
							<Icon size={17} strokeWidth={1.5} aria-hidden="true" class="shrink-0" />
							<span class={labelClass}>{label}</span>
						</Link>
					</li>
				{/each}
				{#each operations as { label, icon: Icon } (label)}
					<li>
						<span
							class={[row, 'text-ink-muted']}
							title={collapsed ? `${label} (planned)` : undefined}
						>
							<Icon size={17} strokeWidth={1.5} aria-hidden="true" class="shrink-0" />
							<span class={labelClass}>{label}<span class="sr-only"> (planned)</span></span>
						</span>
					</li>
				{/each}
			</ul>
		</div>
		<div>
			<p class={[labelClass, 'overflow-hidden px-3 pb-2 text-xs font-medium text-ink-muted']}>
				Account
			</p>
			<ul class="space-y-1">
				<li>
					<Link
						href={customersHref}
						variant="plain"
						aria-current={page.route.id === '/admin/(staff)/customers'
							? 'page'
							: inCustomers
								? 'location'
								: undefined}
						title={collapsed ? 'Customers' : undefined}
						onclick={onNavigate}
						class={[
							row,
							inCustomers
								? 'bg-panel-muted font-semibold text-ink'
								: 'text-ink-muted hover:bg-panel-muted hover:text-ink'
						]}
					>
						<ContactRound size={17} strokeWidth={1.5} aria-hidden="true" class="shrink-0" />
						<span class={labelClass}>Customers</span>
					</Link>
				</li>
				{#if isOwner}
					<li>
						<Link
							href={adminsHref}
							variant="plain"
							aria-current={onAdmins ? 'page' : inAdmins ? 'location' : undefined}
							title={collapsed ? 'Admin accounts' : undefined}
							onclick={onNavigate}
							class={[
								row,
								inAdmins
									? 'bg-panel-muted font-semibold text-ink'
									: 'text-ink-muted hover:bg-panel-muted hover:text-ink'
							]}
						>
							<UsersRound size={17} strokeWidth={1.5} aria-hidden="true" class="shrink-0" />
							<span class={labelClass}>Admin accounts</span>
						</Link>
					</li>
				{/if}
			</ul>
		</div>
	</div>

	<div class="mt-auto shrink-0 space-y-1 pt-6">
		<Link
			href={websiteHref}
			variant="plain"
			title={collapsed ? 'View website' : undefined}
			class={[row, 'text-ink-muted hover:bg-panel-muted hover:text-ink']}
		>
			<ExternalLink size={17} strokeWidth={1.5} aria-hidden="true" class="shrink-0" />
			<span class={labelClass}>View website</span>
			<ArrowUpRight
				size={16}
				strokeWidth={1.5}
				aria-hidden="true"
				class={[labelClass, 'ml-auto shrink-0']}
			/>
		</Link>
		<Link
			href={profileHref}
			variant="plain"
			aria-current={onProfile ? 'page' : inProfile ? 'location' : undefined}
			title={collapsed ? 'Profile' : undefined}
			onclick={onNavigate}
			class={[
				row,
				inProfile
					? 'bg-panel-muted font-semibold text-ink'
					: 'text-ink-muted hover:bg-panel-muted hover:text-ink'
			]}
		>
			<UserRound size={17} strokeWidth={1.5} aria-hidden="true" class="shrink-0" />
			<span class={labelClass}>Profile</span>
		</Link>
		<Button
			onclick={onSignOut}
			disabled={signingOut}
			variant="danger"
			size="custom"
			title={collapsed ? 'Sign out' : undefined}
			class={[row, 'mt-2 w-full justify-start']}
		>
			<LogOut size={17} strokeWidth={1.5} aria-hidden="true" class="shrink-0" />
			<span class={labelClass}>{signingOut ? 'Signing out…' : 'Sign out'}</span>
		</Button>
	</div>
</nav>
