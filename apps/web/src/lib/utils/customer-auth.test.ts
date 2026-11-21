import { describe, expect, it } from 'vitest';
import { customerAccountPath, customerCheckoutSlug, customerReturnPath } from './customer-auth.ts';

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

describe('integrated customer return routes', () => {
	const id = '12345678-1234-4321-8765-123456789abc';
	it.each([
		`/payment/${id}`,
		`/bookings/${id}/success`,
		`/bookings/${id}/expired`,
		`/bookings/${id}/review`,
		`/account/bookings/${id}`,
		`/account/bookings/${id}/invoice`,
		'/checkout/toyota-avanza'
	])('preserves %s', (path) => {
		expect(customerReturnPath(path)).toBe(path);
	});
	it.each([
		'/admin/bookings',
		'//example.invalid',
		'/payment/preview-fake',
		`/bookings/${id}/paid`,
		`/account/bookings/${id}/invoice?paid=true`,
		'/checkout/car%2Fadmin',
		'/checkout/car\n',
		'/checkout/%E0%A4%A'
	])('rejects %s', (path) => {
		expect(customerReturnPath(path)).toBeUndefined();
	});
});

describe('booking history return destinations', () => {
	const id = '12345678-1234-4321-8765-123456789abc';
	it.each(['/account/bookings', `/account/bookings/${id}`, `/account/bookings/${id}/invoice`])(
		'preserves the allowed list context on %s',
		(path) => {
			expect(customerReturnPath(`${path}?status=confirmed&page=2`)).toBe(
				`${path}?status=confirmed&page=2`
			);
		}
	);
	it('normalizes filter/page without allowing arbitrary destinations or query parameters', () => {
		expect(customerReturnPath('/account/bookings?status=invalid&page=-1')).toBe(
			'/account/bookings'
		);
		expect(customerReturnPath('/account/bookings?status=all&page=1')).toBe('/account/bookings');
		expect(customerReturnPath('/account/bookings?next=https://example.invalid')).toBeUndefined();
		expect(customerReturnPath('/account/bookings?status=%E0%A4%A')).toBeUndefined();
		expect(
			customerReturnPath(`/account/bookings/${id}/invoice?status=confirmed#other`)
		).toBeUndefined();
	});
});
