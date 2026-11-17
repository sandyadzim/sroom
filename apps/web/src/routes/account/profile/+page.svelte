<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowRight, LockKeyhole, UserRound } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import Link from '#lib/components/ui/Link.svelte';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { formatWitaDate } from '#lib/utils/date.ts';

	const profile = createQuery(() => customerProfileQuery);
	const name = $derived(profile.data?.full_name.trim());
	const initials = $derived(
		name
			?.split(/\s+/)
			.slice(0, 2)
			.map((part) => Array.from(part)[0])
			.join('')
	);
</script>

<svelte:head>
	<title>Profile — SROOM</title>
</svelte:head>

{#if profile.data}
	<header class="mb-7 sm:mb-9">
		<h1 class="text-4xl font-bold tracking-tight sm:text-5xl">Profile</h1>
		<p class="mt-2 text-sm leading-6 text-ink-muted sm:text-base">
			View your account details and manage your password.
		</p>
	</header>

	<section
		aria-labelledby="personal-details"
		class="overflow-hidden rounded-2xl border border-line"
	>
		<div class="flex items-center gap-4 bg-panel-muted p-5 sm:p-6">
			<div
				class="grid size-12 shrink-0 place-items-center rounded-full bg-primary font-display text-base font-bold text-on-primary uppercase sm:size-14"
				aria-hidden="true"
			>
				{#if initials}{initials}{:else}<UserRound size={22} />{/if}
			</div>
			<div class="min-w-0">
				<h2 id="personal-details" class="text-lg font-bold tracking-tight">Personal details</h2>
				<p class="mt-0.5 text-sm text-ink-muted">
					Member since <time datetime={profile.data.created_at}
						>{formatWitaDate(profile.data.created_at)}</time
					>
				</p>
			</div>
		</div>
		<dl class="divide-y divide-line px-5 sm:px-6">
			<div class="grid gap-1 py-4 sm:grid-cols-[9rem_minmax(0,1fr)] sm:gap-4 sm:py-5">
				<dt class="text-sm text-ink-muted">Full name</dt>
				<dd class="min-w-0 text-sm font-semibold wrap-anywhere">{name || 'Not provided'}</dd>
			</div>
			<div class="grid gap-1 py-4 sm:grid-cols-[9rem_minmax(0,1fr)] sm:gap-4 sm:py-5">
				<dt class="text-sm text-ink-muted">Email address</dt>
				<dd class="min-w-0 text-sm font-semibold wrap-anywhere">{profile.data.email}</dd>
			</div>
		</dl>
	</section>

	<section aria-labelledby="security-title" class="mt-8 sm:mt-10">
		<h2 id="security-title" class="text-xl font-bold tracking-tight">Security</h2>
		<Link
			href={resolve('/account/profile/change-password')}
			variant="plain"
			class="mt-3 flex min-h-14 items-center gap-3 rounded-xl border border-line px-4 py-3 text-sm font-semibold transition-colors hover:bg-panel-muted motion-reduce:transition-none"
		>
			<LockKeyhole size={18} aria-hidden="true" class="text-ink-muted" />
			Change password
			<ArrowRight size={17} aria-hidden="true" class="ml-auto text-ink-muted" />
		</Link>
	</section>
{/if}
