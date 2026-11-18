<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { login } from '#lib/api/identity/auth.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import Wordmark from '#lib/components/layout/Wordmark.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { staffProfileQuery } from '#lib/queries/identity.ts';
	import { validateAuthField, validateAuthForm } from '#lib/validation/auth.ts';

	let email = $state('');
	let password = $state('');
	let message = $state<string>();
	let fields = $state<{ email?: string; password?: string }>({});
	const values = $derived({ email, password, confirmPassword: '' });
	const queryClient = useQueryClient();
	const signIn = createMutation(() => ({
		mutationFn: login
	}));

	function validateField(field: 'email' | 'password', event: 'input' | 'blur') {
		fields[field] = validateAuthField(field, values, event);
	}

	async function submit(event: SubmitEvent & { currentTarget: HTMLFormElement }) {
		event.preventDefault();
		if (signIn.isPending) return;

		const form = event.currentTarget;
		message = undefined;
		fields = validateAuthForm('login', values);
		if (fields.email || fields.password) {
			form
				.querySelector<HTMLInputElement>(fields.email ? '[name=email]' : '[name=password]')
				?.focus();
			return;
		}

		try {
			const staff = await signIn.mutateAsync({ email: email.trim(), password });
			password = '';
			queryClient.removeQueries();
			queryClient.setQueryData(staffProfileQuery.queryKey, staff);
			await goto(resolve('/admin'), { replace: true });
		} catch (cause) {
			const apiError = getApiError(cause);
			fields = { email: apiError?.details?.email, password: apiError?.details?.password };
			message = apiError?.message ?? 'Could not sign in. Please try again.';
			password = '';
		} finally {
			signIn.reset();
		}
	}
</script>

<svelte:head>
	<title>Staff sign in — SROOM</title>
</svelte:head>

<div class="grid min-h-dvh px-6 py-12 sm:place-items-center sm:py-16">
	<section aria-labelledby="staff-login-heading" class="mx-auto w-full max-w-sm">
		<Wordmark />
		<h1 id="staff-login-heading" class="mt-8 font-display text-3xl font-semibold tracking-tight">
			Staff sign in
		</h1>

		<form
			method="post"
			class="mt-7 space-y-5"
			aria-busy={signIn.isPending}
			novalidate
			onsubmit={submit}
		>
			<TextField
				label="Email"
				name="email"
				type="email"
				autocomplete="username"
				autocapitalize="none"
				spellcheck={false}
				bind:value={email}
				error={fields.email}
				oninput={(event) => {
					email = event.currentTarget.value;
					validateField('email', 'input');
				}}
				onblur={() => validateField('email', 'blur')}
				required
			/>
			<TextField
				label="Password"
				name="password"
				type="password"
				autocomplete="current-password"
				bind:value={password}
				error={fields.password}
				oninput={(event) => {
					password = event.currentTarget.value;
					validateField('password', 'input');
				}}
				onblur={() => validateField('password', 'blur')}
				required
			/>
			<div aria-live="polite" aria-atomic="true" class="empty:hidden">
				{#if message}
					<p class="text-sm text-negative">{message}</p>
				{/if}
			</div>
			<Button type="submit" class="w-full" disabled={signIn.isPending}>
				{signIn.isPending ? 'Signing in…' : 'Sign in'}
			</Button>
		</form>
	</section>
</div>
