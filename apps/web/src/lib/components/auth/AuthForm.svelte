<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { onMount, untrack } from 'svelte';
	import { loginCustomer, registerCustomer } from '#lib/api/identity/customer-auth.ts';
	import { getApiError } from '#lib/api/error.ts';
	import { customerProfileQuery } from '#lib/queries/customer-auth.ts';
	import { useToast } from '#lib/ui/toast.ts';
	import {
		customerReturnPath,
		customerAccountPath,
		customerCheckoutSlug
	} from '#lib/utils/customer-auth.ts';
	import { useRegistrationEmail } from './registration-email.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import {
		validateAuthField,
		validateAuthForm,
		type AuthMode,
		type AuthValues,
		type AuthField,
		type AuthErrors
	} from '#lib/validation/auth.ts';

	let { mode }: { mode: AuthMode } = $props();
	const register = $derived(mode === 'register');
	const prefill = useRegistrationEmail();
	const toast = useToast();
	const profile = createQuery(() => customerProfileQuery);
	const queryClient = useQueryClient();
	const signIn = createMutation(() => ({ mutationFn: loginCustomer, retry: false, gcTime: 0 }));
	const signUp = createMutation(() => ({ mutationFn: registerCustomer, retry: false, gcTime: 0 }));
	const next = $derived(page.url.searchParams.get('next'));
	const nextPath = $derived(customerReturnPath(next));
	const destination = $derived.by(() => {
		const account = customerAccountPath(nextPath ?? null);
		if (account) return resolve(account);
		const slug = customerCheckoutSlug(nextPath ?? null);
		if (slug) return resolve('/checkout/[slug]', { slug });
		if (!nextPath) return resolve('/cars');
		const parts = nextPath.split('/');
		if (parts[1] === 'payment') return resolve('/payment/[bookingId]', { bookingId: parts[2] });
		if (parts[1] === 'bookings')
			return resolve('/bookings/[bookingId]/[outcome]', { bookingId: parts[2], outcome: parts[3] });
		return parts[4] === 'invoice'
			? resolve('/account/bookings/[bookingId]/invoice', { bookingId: parts[3] })
			: resolve('/account/bookings/[bookingId]', { bookingId: parts[3] });
	});
	const nextQuery = $derived(nextPath ? (`?next=${encodeURIComponent(nextPath)}` as const) : '');
	const loginHref = $derived(`${resolve('/(auth)/login')}${nextQuery}`);
	const registerHref = $derived(`${resolve('/(auth)/register')}${nextQuery}`);
	let values: AuthValues = $state({
		fullName: '',
		email: untrack(() => (register ? '' : prefill.email)),
		password: '',
		confirmPassword: ''
	});
	untrack(() => {
		if (!register) prefill.email = '';
	});
	let errors: AuthErrors = $state({});
	let message = $state('');
	let navigating = $state(false);
	let ready = $state(false);
	onMount(() => {
		ready = true;
	});
	const submitting = $derived(signIn.isPending || signUp.isPending || navigating);

	$effect(() => {
		if (profile.data && !submitting) {
			navigating = true;
			void goto(destination, { replace: true });
		}
	});

	function handleInput(field: AuthField, value: string) {
		values[field] = value;
		errors[field] = validateAuthField(field, values, 'input', mode);
		if (field === 'password' && values.confirmPassword) {
			errors.confirmPassword = validateAuthField('confirmPassword', values, 'input', mode);
		}
	}

	function focusInvalid(form: HTMLFormElement) {
		const first = [...form.elements].find(
			(element): element is HTMLInputElement =>
				element instanceof HTMLInputElement && Boolean(errors[element.name as AuthField])
		);
		first?.focus();
	}

	async function handleSubmit(event: SubmitEvent & { currentTarget: HTMLFormElement }) {
		event.preventDefault();
		if (!ready || submitting || profile.isLoading || profile.data) return;
		const form = event.currentTarget;
		const submitted = new FormData(form);
		values.email = String(submitted.get('email') ?? '');
		values.password = String(submitted.get('password') ?? '');
		if (register) {
			values.fullName = String(submitted.get('fullName') ?? '');
			values.confirmPassword = String(submitted.get('confirmPassword') ?? '');
		}
		message = '';
		errors = validateAuthForm(mode, values);
		if (Object.values(errors).some(Boolean)) {
			focusInvalid(form);
			return;
		}

		const credentials = { email: values.email.trim(), password: values.password };
		try {
			if (register) {
				const account = await signUp.mutateAsync({
					...credentials,
					full_name: values.fullName?.trim() || undefined
				});
				prefill.email = account.email;
				toast({ kind: 'success', message: 'Account created. Sign in to continue.' });
				navigating = true;
				await goto(resolve(`/(auth)/login${nextQuery}`), { replace: true });
			} else {
				const customer = await signIn.mutateAsync(credentials);
				await queryClient.cancelQueries({ queryKey: customerProfileQuery.queryKey });
				navigating = true;
				queryClient.setQueryData(customerProfileQuery.queryKey, customer);
				await goto(destination, { replace: true });
			}
		} catch (cause) {
			const apiError = getApiError(cause);
			errors = {
				fullName: register ? apiError?.details?.full_name : undefined,
				email:
					register && apiError?.code === 'EMAIL_ALREADY_REGISTERED'
						? 'An account already uses this email.'
						: apiError?.details?.email,
				password: apiError?.details?.password
			};
			if (Object.values(errors).some(Boolean)) {
				focusInvalid(form);
			} else if (apiError?.code === 'INVALID_CREDENTIALS') {
				message = 'Email or password is incorrect.';
			} else if (apiError?.code === 'RATE_LIMITED') {
				message = 'Too many attempts. Wait a moment and try again.';
			} else if (apiError?.code === 'CUSTOMER_REQUIRED') {
				message = 'Use a customer account to sign in here.';
			} else {
				message = register
					? 'Could not create your account. Please try again.'
					: 'Could not sign in. Please try again.';
			}
			navigating = false;
		} finally {
			values.password = '';
			values.confirmPassword = '';
			signIn.reset();
			signUp.reset();
		}
	}
</script>

<section aria-labelledby="auth-heading" class="page-container py-10 sm:py-16">
	<div class="mx-auto w-full max-w-md">
		<h1 id="auth-heading" class="font-display text-3xl font-semibold tracking-tight sm:text-4xl">
			{register ? 'Create your account.' : 'Welcome back.'}
		</h1>
		<p class="mt-3 text-ink-muted">
			{register ? 'Your next drive starts here.' : 'Sign in to plan your next drive.'}
		</p>

		<form
			method="post"
			class="mt-8 space-y-5"
			aria-busy={submitting}
			novalidate
			onsubmit={handleSubmit}
		>
			{#if register}
				<TextField
					label="Full name"
					name="fullName"
					autocomplete="name"
					bind:value={values.fullName}
					error={errors.fullName}
					oninput={(event) => handleInput('fullName', event.currentTarget.value)}
				/>
			{/if}
			<TextField
				label="Email"
				name="email"
				type="email"
				autocomplete="email"
				autocapitalize="none"
				spellcheck={false}
				required
				bind:value={values.email}
				error={errors.email}
				oninput={(event) => handleInput('email', event.currentTarget.value)}
				onblur={() => (errors.email = validateAuthField('email', values, 'blur', mode))}
			/>
			<TextField
				label="Password"
				name="password"
				type="password"
				autocomplete={register ? 'new-password' : 'current-password'}
				required
				bind:value={values.password}
				error={errors.password}
				oninput={(event) => handleInput('password', event.currentTarget.value)}
				onblur={() => (errors.password = validateAuthField('password', values, 'blur', mode))}
			/>
			{#if register}
				<TextField
					label="Confirm password"
					name="confirmPassword"
					type="password"
					autocomplete="new-password"
					required
					bind:value={values.confirmPassword}
					error={errors.confirmPassword}
					oninput={(event) => handleInput('confirmPassword', event.currentTarget.value)}
					onblur={() =>
						(errors.confirmPassword = validateAuthField('confirmPassword', values, 'blur', mode))}
				/>
			{/if}
			<div aria-live="polite" aria-atomic="true" class="empty:hidden">
				{#if message}<p class="text-sm text-negative">{message}</p>{/if}
			</div>
			<Button
				type="submit"
				class="mt-3 w-full"
				disabled={!ready || submitting || profile.isLoading || Boolean(profile.data)}
			>
				{#if submitting}
					{register ? 'Creating account…' : 'Signing in…'}
				{:else}
					{register ? 'Create Account' : 'Sign In'}
				{/if}
			</Button>
		</form>

		<p class="mt-6 text-center text-sm text-ink-muted">
			{register ? 'Already have an account?' : 'New to SROOM?'}
			<Link href={register ? loginHref : registerHref} class="ml-1 font-medium text-ink">
				{register ? 'Sign In' : 'Create an account'}
			</Link>
		</p>
	</div>
</section>
