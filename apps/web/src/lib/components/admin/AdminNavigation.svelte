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
		ExternalLink,
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
	const adminsHref = $derived(resolve('admin/admins'));
	const onAdmins = $derived(page.url.pathname === new URL(adminsHref, page.url.href).pathname);
	const profilePath = $derived(new URL(profileHref, page.url.href).pathname);
	const onProfile = $derived(page.url.pathname === profilePath);
	const inProfile = $derived(onProfile || page.url.pathname.startsWith(`${profilePath}/`));
	const row = 'flex min-h-11 items-center gap-2 overflow-hidden rounded-lg px-3 text-sm';
	const labelClass = $derived([
		'whitespace-nowrap motion-safe:transition-opacity motion-safe:duration-150',
		collapsed && 'opacity-0'
	]);
</script>

<nav class="flex flex-1 flex-col" aria-label="Staff navigation">
	<div class="space-y-6">
		<div>
			<p
				class={[
					labelClass,
					'flex justify-between gap-2 overflow-hidden px-3 pb-2 text-xs font-medium text-ink-muted'
				]}
			>
				Operations <span class="font-normal">Planned</span>
			</p>
			<ul class="space-y-1">
				{#each operations as { label, icon: Icon } (label)}
					<li>
						<span
							class={[row, 'text-ink-muted']}
							title={collapsed ? `${label} (planned)` : undefined}
						>
							<Icon size={17} strokeWidth={1.5} aria-hidden="true" class="shrink-0" />
							<span class={labelClass}>{label}</span>
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
				</li>
				{#if isOwner}
					<li>
						<Link
							href={adminsHref}
							variant="plain"
							aria-current={onAdmins ? 'page' : undefined}
							title={collapsed ? 'Admin accounts' : undefined}
							onclick={onNavigate}
							class={[
								row,
								onAdmins
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

	<div class="mt-auto space-y-1 pt-6">
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
		<Button
			onclick={onSignOut}
			disabled={signingOut}
			variant="ghost"
			size="custom"
			title={collapsed ? 'Sign out' : undefined}
			class={[row, 'w-full justify-start font-normal']}
		>
			<LogOut size={17} strokeWidth={1.5} aria-hidden="true" class="shrink-0" />
			<span class={labelClass}>{signingOut ? 'Signing out…' : 'Sign out'}</span>
		</Button>
	</div>
</nav>
