<script lang="ts">
	import { resolve } from '$app/paths';
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
	let values: AuthValues = $state({ fullName: '', email: '', password: '', confirmPassword: '' });
	let errors: AuthErrors = $state({});

	function handleInput(field: AuthField, value: string) {
		values[field] = value;
		errors[field] = validateAuthField(field, values, 'input');
		if (field === 'password' && values.confirmPassword) {
			errors.confirmPassword = validateAuthField('confirmPassword', values, 'input');
		}
	}

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		errors = validateAuthForm(mode, values);
		if (Object.values(errors).some(Boolean)) {
			const firstInvalid = Object.keys(errors)[0];
			const form = event.currentTarget as HTMLFormElement;
			(form.elements.namedItem(firstInvalid) as HTMLInputElement)?.focus();
			return;
		}
		// UI-only handoff: never log credentials or create a simulated session.
		console.log(`${register ? 'Register' : 'Login'} form submitted. Validation passed.`);
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

		<form method="post" class="mt-8 space-y-5" novalidate onsubmit={handleSubmit}>
			{#if register}
				<TextField
					label="Full name"
					name="fullName"
					autocomplete="name"
					bind:value={values.fullName}
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
				value={values.email}
				error={errors.email}
				oninput={(event) => handleInput('email', event.currentTarget.value)}
				onblur={() => (errors.email = validateAuthField('email', values, 'blur'))}
			/>
			<TextField
				label="Password"
				name="password"
				type="password"
				autocomplete={register ? 'new-password' : 'current-password'}
				required
				value={values.password}
				error={errors.password}
				oninput={(event) => handleInput('password', event.currentTarget.value)}
				onblur={() => (errors.password = validateAuthField('password', values, 'blur'))}
			/>
			{#if register}
				<TextField
					label="Confirm password"
					name="confirmPassword"
					type="password"
					autocomplete="new-password"
					required
					value={values.confirmPassword}
					error={errors.confirmPassword}
					oninput={(event) => handleInput('confirmPassword', event.currentTarget.value)}
					onblur={() =>
						(errors.confirmPassword = validateAuthField('confirmPassword', values, 'blur'))}
				/>
			{/if}
			<Button type="submit" class="mt-3 w-full">{register ? 'Create Account' : 'Sign In'}</Button>
		</form>

		<p class="mt-6 text-center text-sm text-ink-muted">
			{register ? 'Already have an account?' : 'New to SROOM?'}
			<Link
				href={register ? resolve('/(auth)/login') : resolve('/(auth)/register')}
				class="ml-1 font-medium text-ink"
			>
				{register ? 'Sign In' : 'Create an account'}
			</Link>
		</p>
	</div>
</section>
