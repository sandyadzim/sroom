import { describe, expect, it, vi } from 'vitest';
vi.mock('$app/env/public', () => ({ PUBLIC_GATEWAY_BASE_URL: 'http://localhost:8080' }));
import { customerBookingsQuery } from './rental.ts';
describe('customer booking list query', () => {
	it('isolates accounts, filters and pages and waits for a verified customer', () => {
		const base = customerBookingsQuery('a', { status: 'all', page: 1 });
		expect(base.queryKey).not.toEqual(
			customerBookingsQuery('b', { status: 'all', page: 1 }).queryKey
		);
		expect(base.queryKey).not.toEqual(
			customerBookingsQuery('a', { status: 'confirmed', page: 1 }).queryKey
		);
		expect(base.queryKey).not.toEqual(
			customerBookingsQuery('a', { status: 'all', page: 2 }).queryKey
		);
		expect(base.queryKey.at(-1)).toEqual({ status: undefined, page: 1, limit: 5 });
		expect(customerBookingsQuery('', { status: 'all', page: 1 }).enabled).toBe(false);
		expect(base.retry).toBe(false);
	});
});
