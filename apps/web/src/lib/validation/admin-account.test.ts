import { describe, expect, it } from 'vitest';
import type { AdminAccount, CreateAdminInput } from '#lib/types/identity.ts';
import { buildAdminUpdate, validateAdminField } from './admin-account.ts';

const account: AdminAccount = {
	id: 'fixture-admin',
	full_name: 'Alex Morgan',
	email: 'admin@example.test',
	role: 'ADMIN',
	status: 'active',
	created_at: '2026-11-14T00:00:00Z',
	updated_at: '2026-11-14T00:00:00Z'
};

const values: CreateAdminInput = {
	full_name: account.full_name,
	email: account.email,
	status: account.status,
	password: ''
};

describe('admin account edit payload', () => {
	it('omits unchanged fields and an empty password', () => {
		expect(buildAdminUpdate(account, { ...values, status: 'inactive' })).toEqual({
			status: 'inactive'
		});
	});

	it('sends an empty name when the owner clears it', () => {
		expect(buildAdminUpdate(account, { ...values, full_name: '  ' })).toEqual({ full_name: '' });
	});

	it('omits equivalent name and email values after normalization', () => {
		expect(
			buildAdminUpdate(account, {
				...values,
				full_name: ' Alex Morgan ',
				email: ' ADMIN@EXAMPLE.TEST '
			})
		).toEqual({});
	});

	it('preserves the exact password entered', () => {
		expect(buildAdminUpdate(account, { ...values, password: ' pass 123 ' })).toEqual({
			password: ' pass 123 '
		});
	});
});

describe('admin account validation', () => {
	it('requires a password on create and permits a blank password on edit', () => {
		expect(validateAdminField('password', '', true, 'input')).toBeUndefined();
		expect(validateAdminField('password', '', true)).toBe('Password is required.');
		expect(validateAdminField('password', '', false)).toBeUndefined();
		expect(validateAdminField('password', 'short', false)).toBe('Use at least 8 characters.');
	});

	it('checks email format while typing and waits for blur to show required', () => {
		expect(validateAdminField('email', '  ', true, 'input')).toBeUndefined();
		expect(validateAdminField('email', '  ', true)).toBe('Email is required.');
		expect(validateAdminField('email', 'admin@', true, 'input')).toBe(
			'Enter a valid email address.'
		);
		expect(validateAdminField('email', ' admin@example.test ', true, 'input')).toBeUndefined();
	});
});
