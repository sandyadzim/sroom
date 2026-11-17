<script lang="ts">
	import { useQueryClient } from '@tanstack/svelte-query';
	import { changeCustomerPassword } from '#lib/api/identity/customer-auth.ts';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import ChangePasswordForm from '#lib/components/auth/ChangePasswordForm.svelte';

	const queryClient = useQueryClient();

	async function expireSession() {
		await queryClient.cancelQueries({ queryKey: customerProfileQuery.queryKey });
		queryClient.setQueryData(customerProfileQuery.queryKey, null);
	}
</script>

<svelte:head>
	<title>Change password — SROOM</title>
</svelte:head>

<h1 class="text-3xl font-bold tracking-tight sm:text-4xl">Change password</h1>
<p class="mt-2 text-sm leading-6 text-ink-muted sm:text-base">
	Enter your current password to set a new one.
</p>

<ChangePasswordForm
	profileRoute="/account/profile"
	changePassword={changeCustomerPassword}
	onSessionExpired={expireSession}
/>
