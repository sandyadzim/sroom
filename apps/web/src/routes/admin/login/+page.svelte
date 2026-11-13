<script lang="ts">
	import { enhance } from '$app/forms';
	import { tick } from 'svelte';
	import Wordmark from '#lib/components/layout/Wordmark.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { validateAuthField, validateAuthForm } from '#lib/validation/auth.ts';
	import type { PageProps, SubmitFunction } from './$types';

	let { form }: PageProps = $props();
	let pending = $state(false);
	let inlineErrors = $state<{ email?: string | null; password?: string | null }>({});

	function loginValues(fields: FormData) {
		return {
			email: String(fields.get('email') ?? ''),
			password: String(fields.get('password') ?? ''),
			confirmPassword: ''
		};
	}

	function validateField(
		input: HTMLInputElement,
		field: 'email' | 'password',
		event: 'input' | 'blur'
	) {
		if (!input.form) return;
		inlineErrors[field] =
			validateAuthField(field, loginValues(new FormData(input.form)), event) ?? null;
	}

	const submit: SubmitFunction = ({ cancel, formElement, formData }) => {
		if (pending) return cancel();
		const errors = validateAuthForm('login', loginValues(formData));
		inlineErrors = { email: errors.email ?? null, password: errors.password ?? null };
		if (errors.email || errors.password) {
			cancel();
			const field = formElement.elements.namedItem(errors.email ? 'email' : 'password');
			if (field instanceof HTMLInputElement) void tick().then(() => field.focus());
			return;
		}

		pending = true;
		return async ({ result, update }) => {
			try {
				await update();
			} finally {
				pending = false;
			}
			if (result.type === 'failure') {
				inlineErrors = {};
				await tick();
				const password = formElement.querySelector<HTMLInputElement>('input[name="password"]');
				if (password) password.value = '';
				const target = formElement.querySelector<HTMLElement>(
					'input[aria-invalid="true"], [data-login-error]'
				);
				target?.focus();
			}
		};
	};
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

		<form method="post" class="mt-7 space-y-5" aria-busy={pending} novalidate use:enhance={submit}>
			<TextField
				label="Email"
				name="email"
				type="email"
				autocomplete="username"
				autocapitalize="none"
				spellcheck={false}
				value={form?.email ?? ''}
				error={inlineErrors.email === undefined
					? form?.fields?.email
					: (inlineErrors.email ?? undefined)}
				oninput={(event) => validateField(event.currentTarget, 'email', 'input')}
				onblur={(event) => validateField(event.currentTarget, 'email', 'blur')}
				required
			/>
			<TextField
				label="Password"
				name="password"
				type="password"
				autocomplete="current-password"
				error={inlineErrors.password === undefined
					? form?.fields?.password
					: (inlineErrors.password ?? undefined)}
				oninput={(event) => validateField(event.currentTarget, 'password', 'input')}
				onblur={(event) => validateField(event.currentTarget, 'password', 'blur')}
				required
			/>
			<div aria-live="polite" aria-atomic="true" class="empty:hidden">
				{#if form?.message}
					<p class="text-sm text-negative" tabindex="-1" data-login-error>{form.message}</p>
				{/if}
			</div>
			<Button type="submit" class="w-full" disabled={pending}>
				{pending ? 'Signing in…' : 'Sign in'}
			</Button>
		</form>
	</section>
</div>
