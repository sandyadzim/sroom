import { z } from 'zod';
import type { AdminAccount, CreateAdminInput, UpdateAdminInput } from '#lib/types/identity.ts';

export const adminAccountFields = ['full_name', 'email', 'status', 'password'] as const;
export type AdminAccountField = (typeof adminAccountFields)[number];
export type AdminAccountErrors = Partial<Record<AdminAccountField, string>>;

const emailFormat = z.email();

export function normalizeAdminInput(values: CreateAdminInput): CreateAdminInput {
	return {
		...values,
		full_name: values.full_name.trim(),
		email: values.email.trim().toLowerCase()
	};
}

export function validateAdminField(
	field: AdminAccountField,
	value: string,
	isCreate: boolean,
	event: 'input' | 'blur' = 'blur'
): string | undefined {
	switch (field) {
		case 'full_name':
			if ([...value.trim()].length > 100) return 'Use at most 100 characters.';
			break;
		case 'email': {
			const email = value.trim();
			if (!email) return event === 'blur' ? 'Email is required.' : undefined;
			if (email.length > 255) return 'Use at most 255 characters.';
			if (!emailFormat.safeParse(email).success) return 'Enter a valid email address.';
			break;
		}
		case 'status':
			if (value !== 'active' && value !== 'inactive') return 'Select an account status.';
			break;
		case 'password':
			if (!value) return isCreate && event === 'blur' ? 'Password is required.' : undefined;
			if ([...value].length < 8) return 'Use at least 8 characters.';
			break;
	}
}

export function buildAdminUpdate(
	account: AdminAccount,
	values: CreateAdminInput
): UpdateAdminInput {
	const { full_name, email, status, password } = normalizeAdminInput(values);
	const update: UpdateAdminInput = {};
	if (full_name !== account.full_name.trim()) update.full_name = full_name;
	if (email !== account.email.trim().toLowerCase()) update.email = email;
	if (status !== account.status) update.status = status;
	if (password) update.password = password;
	return update;
}
