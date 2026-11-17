import { describe, expect, it } from 'vitest';
import { customerCheckoutSlug } from './customer-auth.ts';

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
