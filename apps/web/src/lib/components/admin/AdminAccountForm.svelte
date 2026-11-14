<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { untrack } from 'svelte';
	import {
		adminAccountsEndpoint,
		createAdminAccount,
		updateAdminAccount
	} from '#lib/api/identity/admin.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { adminAccountQuery } from '#lib/queries/identity.ts';
	import type { AdminAccount, CreateAdminInput } from '#lib/types/identity.ts';
	import { useToast } from '#lib/ui/toast.ts';
	import {
		adminAccountFields,
		buildAdminUpdate,
		normalizeAdminInput,
		validateAdminField,
		type AdminAccountErrors,
		type AdminAccountField
	} from '#lib/validation/admin-account.ts';

	type Props = { mode: 'create'; account?: never } | { mode: 'edit'; account: AdminAccount };
	let { mode, account }: Props = $props();
	const isCreate = $derived(mode === 'create');
	const cancelHref = $derived(
		account ? resolve('/admin/(staff)/admins/[id]', { id: account.id }) : resolve('admin/admins')
	);
	const original = untrack(() => account);
	let values = $state<CreateAdminInput>({
		full_name: original?.full_name ?? '',
		email: original?.email ?? '',
		status: original?.status ?? 'active',
		password: ''
	});
	let fields = $state<AdminAccountErrors>({});
	let statusTrigger = $state<HTMLButtonElement | null>(null);
	const statuses = [
		{ value: 'active', label: 'Active' },
		{ value: 'inactive', label: 'Inactive' }
	];
	const queryClient = useQueryClient();
	const showToast = useToast();
	const saveAdmin = createMutation(() => ({
		mutationFn: (input: CreateAdminInput) =>
			original
				? updateAdminAccount(original.id, buildAdminUpdate(original, input))
				: createAdminAccount(normalizeAdminInput(input)),
		onSuccess: async (saved) => {
			const { queryKey } = adminAccountQuery(saved.id);
			await queryClient.cancelQueries({ queryKey, exact: true });
			queryClient.setQueryData(queryKey, saved);
			await queryClient.invalidateQueries({
				queryKey: [adminAccountsEndpoint],
				refetchType: 'none'
			});
		}
	}));
	const hasChanges = $derived(
		!original || Object.keys(buildAdminUpdate(original, values)).length > 0
	);

	function updateText(field: Exclude<AdminAccountField, 'status'>, value: string) {
		values[field] = value;
		fields[field] = validateAdminField(field, value, isCreate, 'input');
	}

	function validateField(field: AdminAccountField) {
		// Keep a server field error until the owner edits that field.
		fields[field] ??= validateAdminField(field, values[field], isCreate);
	}

	function focusField(form: HTMLFormElement, field: AdminAccountField) {
		if (field === 'status') {
			statusTrigger?.focus();
		} else {
			form.querySelector<HTMLInputElement>(`input[name="${field}"]`)?.focus();
		}
	}

	async function submit(event: SubmitEvent & { currentTarget: HTMLFormElement }) {
		event.preventDefault();
		if (saveAdmin.isPending || !hasChanges) return;
		const form = event.currentTarget;
		fields = {};
		for (const field of adminAccountFields) {
			fields[field] = validateAdminField(field, values[field], isCreate);
		}
		const invalidField = adminAccountFields.find((field) => fields[field]);
		if (invalidField) {
			showToast({ kind: 'warning', message: 'Check the highlighted fields and try again.' });
			focusField(form, invalidField);
			return;
		}

		try {
			await saveAdmin.mutateAsync({ ...values });
		} catch (cause) {
			const error = getApiError(cause);
			for (const field of adminAccountFields) {
				fields[field] = error?.details?.[field];
			}
			if (error?.code === 'EMAIL_ALREADY_REGISTERED') {
				fields.email = error.message;
			}
			showToast({
				kind: 'error',
				message: error?.message ?? 'Could not save this admin account. Please try again.'
			});
			return;
		} finally {
			saveAdmin.reset();
		}

		values.password = '';
		showToast({
			kind: 'success',
			message: isCreate ? 'Admin account created.' : 'Admin account updated.'
		});
		await goto(resolve('admin/admins'));
	}
</script>

<form
	method="post"
	novalidate
	aria-busy={saveAdmin.isPending}
	class="mt-6 max-w-lg border-t border-line pt-6"
	onsubmit={submit}
>
	<div class="space-y-5">
		<TextField
			label="Full name"
			name="full_name"
			autocomplete="off"
			value={values.full_name}
			error={fields.full_name}
			disabled={saveAdmin.isPending}
			oninput={(event) => updateText('full_name', event.currentTarget.value)}
			onblur={() => validateField('full_name')}
		/>
		<TextField
			label="Email"
			name="email"
			type="email"
			autocomplete="off"
			value={values.email}
			error={fields.email}
			disabled={saveAdmin.isPending}
			oninput={(event) => updateText('email', event.currentTarget.value)}
			onblur={() => validateField('email')}
			required
		/>
		<SelectField
			bind:ref={statusTrigger}
			label="Status"
			name="status"
			options={statuses}
			value={values.status}
			error={fields.status}
			disabled={saveAdmin.isPending}
			help={values.status === 'inactive'
				? 'This admin can’t sign in. Existing sessions stay active until they expire or the admin signs out.'
				: undefined}
			onchange={(value) => {
				if (value === 'active' || value === 'inactive') values.status = value;
				fields.status = validateAdminField('status', value, isCreate);
			}}
		/>
		<TextField
			label={isCreate ? 'Password' : 'New password'}
			name="password"
			type="password"
			revealPassword
			autocomplete="new-password"
			value={values.password}
			error={fields.password}
			help={fields.password
				? undefined
				: isCreate
					? 'Use at least 8 characters.'
					: 'Leave blank to keep the current password.'}
			disabled={saveAdmin.isPending}
			oninput={(event) => updateText('password', event.currentTarget.value)}
			onblur={() => validateField('password')}
			required={isCreate}
		/>
	</div>

	<div class="mt-6 flex flex-wrap items-center gap-3">
		<Button type="submit" disabled={saveAdmin.isPending || !hasChanges}>
			{#if saveAdmin.isPending}
				{isCreate ? 'Creating admin…' : 'Saving changes…'}
			{:else}
				{isCreate ? 'Create admin' : 'Save changes'}
			{/if}
		</Button>
		<Button href={cancelHref} variant="ghost">Cancel</Button>
	</div>
</form>
