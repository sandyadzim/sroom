<script lang="ts">
	import { resolve } from '$app/paths';
	import Button from '#lib/components/ui/Button.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const fields = $derived([
		{ label: 'Full name', value: data.profile.full_name.trim() || '—' },
		{ label: 'Email', value: data.profile.email },
		{ label: 'Role', value: data.profile.role === 'OWNER' ? 'Owner' : 'Admin' }
	]);
</script>

<svelte:head>
	<title>Staff profile — SROOM</title>
</svelte:head>

<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Profile</h1>

<section
	aria-labelledby="account-details"
	class="mt-8 grid gap-4 border-t border-line pt-6 lg:grid-cols-[minmax(12rem,1fr)_minmax(0,2fr)] lg:gap-8"
>
	<h2 id="account-details" class="text-lg/7 font-semibold">Account details</h2>

	<dl class="min-w-0 divide-y divide-line rounded-xl bg-page px-4 sm:px-5">
		{#each fields as { label, value } (label)}
			<div class="grid gap-1 py-4 text-sm/6 sm:grid-cols-[8rem_minmax(0,1fr)] sm:gap-4">
				<dt class="text-ink-muted">{label}</dt>
				<dd class="font-medium wrap-anywhere">{value}</dd>
			</div>
		{/each}
	</dl>
</section>

<section
	aria-labelledby="password-heading"
	class="mt-8 grid items-start gap-4 border-t border-line pt-6 lg:grid-cols-[minmax(12rem,1fr)_minmax(0,2fr)] lg:gap-8"
>
	<h2 id="password-heading" class="text-lg/7 font-semibold">Password</h2>
	<div>
		<Button href={resolve('admin/profile/change-password')} variant="secondary">
			Change password
		</Button>
	</div>
</section>
