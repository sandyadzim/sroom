import { describe, expect, it } from 'vitest';
import { loginSchema, registerSchema, validateAuthField, validateAuthForm } from './auth.ts';

const empty = { fullName: '', email: '', password: '', confirmPassword: '' };
const valid = { email: 'driver@example.com', password: 'password', confirmPassword: 'password' };

describe('authentication schemas', () => {
	it('accepts a login without registration fields or additional password rules', () => {
		expect(loginSchema.safeParse({ email: valid.email, password: 'a' }).success).toBe(true);
	});

	it.each([undefined, '', 'A name with no additional restrictions'])(
		'allows optional full name %s',
		(fullName) => {
			expect(registerSchema.safeParse({ ...valid, fullName }).success).toBe(true);
		}
	);

	it('trims email whitespace without changing either password', () => {
		const result = registerSchema.parse({
			...valid,
			email: ` ${valid.email} `,
			password: ' abcdef ',
			confirmPassword: ' abcdef '
		});
		expect(result).toMatchObject({
			email: valid.email,
			password: ' abcdef ',
			confirmPassword: ' abcdef '
		});
	});

	it('checks every required field on empty submit with required messages taking precedence', () => {
		expect(validateAuthForm('register', empty)).toEqual({
			email: 'Email is required.',
			password: 'Password is required.',
			confirmPassword: 'Please confirm your password.'
		});
	});

	it('checks email shape and matching passwords on submit', () => {
		expect(
			validateAuthForm('register', { ...valid, email: 'driver@', confirmPassword: 'b' })
		).toEqual({
			email: 'Enter a valid email address.',
			confirmPassword: 'Passwords do not match.'
		});
	});

	it('shows the eight-character minimum on registration without changing login rules', () => {
		const values = { ...valid, password: 'short', confirmPassword: 'short' };
		expect(validateAuthField('password', values, 'blur', 'register')).toBe(
			'Use at least 8 characters.'
		);
		expect(validateAuthForm('login', values)).toEqual({});
	});

	it('does not require password confirmation for login', () => {
		expect(validateAuthForm('login', { ...valid, confirmPassword: '' })).toEqual({});
	});
});

describe('field validation timing', () => {
	it.each(['email', 'password', 'confirmPassword'] as const)(
		'%s waits until blur to show required',
		(field) => {
			expect(validateAuthField(field, empty, 'input')).toBeUndefined();
			expect(validateAuthField(field, empty, 'blur')).toBeTruthy();
		}
	);

	it('treats a whitespace-only email as empty', () => {
		const values = { ...empty, email: '   ' };
		expect(validateAuthField('email', values, 'input')).toBeUndefined();
		expect(validateAuthField('email', values, 'blur')).toBe('Email is required.');
	});

	it('checks email format immediately on input and clears it when corrected', () => {
		expect(validateAuthField('email', { ...valid, email: 'driver' }, 'input')).toBe(
			'Enter a valid email address.'
		);
		expect(validateAuthField('email', valid, 'input')).toBeUndefined();
	});

	it('checks confirmation immediately while typing, including a space character', () => {
		expect(validateAuthField('confirmPassword', { ...valid, confirmPassword: 'b' }, 'input')).toBe(
			'Passwords do not match.'
		);
		expect(validateAuthField('confirmPassword', { ...valid, confirmPassword: ' ' }, 'input')).toBe(
			'Passwords do not match.'
		);
		expect(validateAuthField('confirmPassword', valid, 'input')).toBeUndefined();
	});

	it('invalidates a previously matching confirmation when the original password changes', () => {
		expect(validateAuthField('confirmPassword', valid, 'input')).toBeUndefined();
		expect(validateAuthField('confirmPassword', { ...valid, password: 'changed' }, 'input')).toBe(
			'Passwords do not match.'
		);
		expect(validateAuthField('confirmPassword', { ...valid, password: '' }, 'input')).toBe(
			'Passwords do not match.'
		);
		expect(validateAuthField('confirmPassword', valid, 'input')).toBeUndefined();
	});
});
