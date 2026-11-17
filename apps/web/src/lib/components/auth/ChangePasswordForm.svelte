<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createMutation } from '@tanstack/svelte-query';
	import { onMount } from 'svelte';
	import type { ChangePasswordInput } from '#lib/types/identity.ts';
	import { getApiError } from '#lib/api/error.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { useToast } from '#lib/ui/toast.ts';
	import {
		readPasswordForm,
		validatePasswordField,
		validatePasswordForm,
		type PasswordField,
		type PasswordErrors
	} from '#lib/validation/change-password.ts';

	let {
		profileRoute,
		changePassword,
		onSessionExpired
	}: {
		profileRoute: '/account/profile' | '/admin/(staff)/profile';
		changePassword: (input: ChangePasswordInput) => Promise<void>;
		onSessionExpired?: () => void;
	} = $props();

	let ready = $state(false);
	onMount(() => {
		ready = true;
	});
	const showToast = useToast();
	const updatePassword = createMutation(() => ({
		mutationFn: changePassword,
		retry: false,
		gcTime: 0
	}));
	let fields = $state<PasswordErrors>({});

	function validateField(input: HTMLInputElement, field: PasswordField, event: 'input' | 'blur') {
		if (!input.form) return;
		const values = readPasswordForm(new FormData(input.form));
		fields[field] = validatePasswordField(field, values, event);
		if (field === 'new_password' && values.confirm_password) {
			fields.confirm_password = validatePasswordField('confirm_password', values, 'input');
		}
	}

	async function submit(event: SubmitEvent & { currentTarget: HTMLFormElement }) {
		event.preventDefault();
		if (!ready || updatePassword.isPending) return;
		const form = event.currentTarget;
		const values = readPasswordForm(new FormData(form));
		fields = validatePasswordForm(values);
		const invalidField = Object.keys(fields).find((field) => fields[field as PasswordField]);
		if (invalidField) {
			showToast({ kind: 'warning', message: 'Check the highlighted fields and try again.' });
			form.querySelector<HTMLInputElement>(`[name="${invalidField}"]`)?.focus();
			return;
		}

		try {
			await updatePassword.mutateAsync({
				current_password: values.current_password,
				new_password: values.new_password
			});
			form.reset();
			showToast({ kind: 'success', message: 'Your password has been updated.' });
			await goto(resolve(profileRoute));
		} catch (cause) {
			const apiError = getApiError(cause);
			if (apiError?.code === 'UNAUTHENTICATED' && onSessionExpired) {
				form.reset();
				showToast({ kind: 'warning', message: 'Your session expired. Please sign in again.' });
				onSessionExpired();
				return;
			}
			fields = {
				current_password:
					apiError?.code === 'INVALID_CURRENT_PASSWORD'
						? apiError.message
						: apiError?.details?.current_password,
				new_password: apiError?.details?.new_password
			};
			showToast({
				kind: 'error',
				message:
					apiError?.code === 'VALIDATION_ERROR'
						? 'Check the highlighted fields and try again.'
						: (apiError?.message ?? 'Could not change your password. Please try again.')
			});
		} finally {
			updatePassword.reset();
		}
	}
</script>

<form
	method="post"
	novalidate
	aria-busy={updatePassword.isPending}
	class="mt-6 max-w-lg border-t border-line pt-6"
	onsubmit={submit}
>
	<div class="space-y-5">
		<TextField
			label="Current password"
			name="current_password"
			type="password"
			revealPassword
			autocomplete="current-password"
			error={fields.current_password}
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
			help={fields.new_password ? undefined : 'Use at least 8 characters.'}
			error={fields.new_password}
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
			error={fields.confirm_password}
			oninput={(event) => validateField(event.currentTarget, 'confirm_password', 'input')}
			onblur={(event) => validateField(event.currentTarget, 'confirm_password', 'blur')}
			required
		/>
	</div>

	<div class="mt-6 flex flex-wrap items-center justify-end gap-2 border-t border-line pt-4">
		<Button href={resolve(profileRoute)} variant="secondary">Cancel</Button>
		<Button type="submit" disabled={!ready || updatePassword.isPending}>
			{updatePassword.isPending ? 'Updating password…' : 'Update password'}
		</Button>
	</div>
</form>
