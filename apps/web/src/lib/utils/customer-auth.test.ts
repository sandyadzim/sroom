import { describe, expect, it } from 'vitest';
import { customerAccountPath, customerCheckoutSlug } from './customer-auth.ts';

describe('customer return destination', () => {
	it('accepts a checkout slug carried as an encoded query parameter', () => {
		const query = new URLSearchParams('next=%2Fcheckout%2Ftoyota-avanza-1-3-e-mt');
		expect(customerCheckoutSlug(query.get('next'))).toBe('toyota-avanza-1-3-e-mt');
	});

	it.each([
		null,
		'https://example.invalid/checkout/car',
		'//example.invalid/checkout/car',
		'/admin/profile',
		'/checkout/../admin',
		'/checkout/car%2fadmin',
		'/checkout/car?other=1',
		'/checkout/car\n',
		'/checkout/some--car'
	])('ignores an unapproved next value of %s', (next) => {
		expect(customerCheckoutSlug(next)).toBeUndefined();
	});
});

describe('customer account return path', () => {
	it.each(['/account/profile', '/account/profile/change-password'])('accepts %s', (path) => {
		expect(customerAccountPath(path)).toBe(path);
	});

	it.each([
		null,
		'/admin/profile',
		'//example.invalid/account/profile',
		'/account/profile?next=/admin',
		'/account/profile/%2e%2e',
		'/account/bookings'
	])('ignores an unapproved path of %s', (path) => {
		expect(customerAccountPath(path)).toBeUndefined();
	});
});
