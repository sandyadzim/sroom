<script lang="ts">
	import { resolve } from '$app/paths';
	import AdminDetailRow from '#lib/components/admin/AdminDetailRow.svelte';
	import AdminDetailSection from '#lib/components/admin/AdminDetailSection.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { staffProfileQuery } from '#lib/queries/identity.ts';

	const profile = createQuery(() => staffProfileQuery);

	const fields = $derived(
		profile.data
			? [
					{ label: 'Full name', value: profile.data.full_name.trim() || '—' },
					{ label: 'Email', value: profile.data.email },
					{ label: 'Role', value: profile.data.role === 'OWNER' ? 'Owner' : 'Admin' }
				]
			: []
	);
</script>

<svelte:head>
	<title>Staff profile — SROOM</title>
</svelte:head>

<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Profile</h1>

<AdminDetailSection title="Account details">
	{#each fields as { label, value } (label)}
		<AdminDetailRow {label}>{value}</AdminDetailRow>
	{/each}
</AdminDetailSection>

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
