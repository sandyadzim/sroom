<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';
	import { tick } from 'svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { useToast } from '#lib/ui/toast.ts';
	import {
		readPasswordForm,
		validatePasswordField,
		validatePasswordForm,
		type PasswordField
	} from '#lib/validation/change-password.ts';
	import type { PageProps, SubmitFunction } from './$types';

	let { form }: PageProps = $props();
	const profileHref = $derived(resolve('admin/profile'));
	const showToast = useToast();
	let pending = $state(false);
	let inlineErrors = $state<Partial<Record<PasswordField, string | null>>>({});

	function validateField(input: HTMLInputElement, field: PasswordField, event: 'input' | 'blur') {
		if (!input.form) return;
		const values = readPasswordForm(new FormData(input.form));
		if (event === 'blur' && inlineErrors[field] === undefined && form?.fields?.[field]) return;
		inlineErrors[field] = validatePasswordField(field, values, event) ?? null;
		if (field === 'new_password' && values.confirm_password) {
			inlineErrors.confirm_password =
				validatePasswordField('confirm_password', values, 'input') ?? null;
		}
	}

	function fieldError(field: PasswordField) {
		return inlineErrors[field] === undefined
			? form?.fields?.[field]
			: (inlineErrors[field] ?? undefined);
	}

	const submit: SubmitFunction = ({ cancel, formElement, formData }) => {
		if (pending) return cancel();
		const errors = validatePasswordForm(readPasswordForm(formData));
		inlineErrors = {
			current_password: errors.current_password ?? null,
			new_password: errors.new_password ?? null,
			confirm_password: errors.confirm_password ?? null
		};
		if (Object.values(errors).some(Boolean)) {
			cancel();
			showToast({ kind: 'warning', message: 'Check the highlighted fields and try again.' });
			void tick().then(() =>
				formElement.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus()
			);
			return;
		}

		pending = true;
		return async ({ result, update }) => {
			try {
				await update({ invalidateAll: false });
				inlineErrors = {};
			} finally {
				pending = false;
			}
			await tick();
			if (result.type === 'failure') {
				const field = formElement.querySelector<HTMLElement>('[aria-invalid="true"]');
				(field ?? formElement.querySelector('button[type="submit"]'))?.focus();
			}
		};
	};
</script>

<svelte:head>
	<title>Change password — SROOM</title>
</svelte:head>

<Link href={profileHref} class="gap-2 text-sm font-medium text-ink-muted hover:text-ink">
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" />
	Back to profile
</Link>
<h1 class="mt-2 text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Change password</h1>

<form
	method="post"
	novalidate
	aria-busy={pending}
	class="mt-6 max-w-lg border-t border-line pt-6"
	use:enhance={submit}
>
	<div class="space-y-5">
		<TextField
			label="Current password"
			name="current_password"
			type="password"
			revealPassword
			autocomplete="current-password"
			error={fieldError('current_password')}
			oninput={(event) => validateField(event.currentTarget, 'current_password', 'input')}
			onblur={(event) => validateField(event.currentTarget, 'current_password', 'blur')}
			required
		/>
		<TextField
			label="New password"
			name="new_password"
			type="password"
			revealPassword
			autocomplete="new-password"
			help={fieldError('new_password') ? undefined : 'Use at least 8 characters.'}
			error={fieldError('new_password')}
			oninput={(event) => validateField(event.currentTarget, 'new_password', 'input')}
			onblur={(event) => validateField(event.currentTarget, 'new_password', 'blur')}
			required
		/>
		<TextField
			label="Confirm new password"
			name="confirm_password"
			type="password"
			revealPassword
			autocomplete="new-password"
			error={fieldError('confirm_password')}
			oninput={(event) => validateField(event.currentTarget, 'confirm_password', 'input')}
			onblur={(event) => validateField(event.currentTarget, 'confirm_password', 'blur')}
			required
		/>
	</div>

	<div class="mt-6 flex flex-wrap items-center gap-3">
		<Button type="submit" disabled={pending}>
			{pending ? 'Updating password…' : 'Update password'}
		</Button>
		<Button href={profileHref} variant="ghost">Cancel</Button>
	</div>
</form>
