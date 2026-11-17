import { describe, expect, it } from 'vitest';
import { getPagination } from './pagination.ts';

describe('catalogue pagination', () => {
	it('covers all eight distinct variants over pages of six without duplicates or omissions', () => {
		const variantIds = Array.from({ length: 8 }, (_, index) => index + 1);
		const first = getPagination(variantIds.length, 1, 6);
		const last = getPagination(variantIds.length, 2, 6);
		const ids = [
			...variantIds.slice(first.offset, first.end),
			...variantIds.slice(last.offset, last.end)
		];
		expect(first).toMatchObject({ start: 1, end: 6, totalPages: 2 });
		expect(last).toMatchObject({ start: 7, end: 8, totalPages: 2 });
		expect(ids).toHaveLength(8);
		expect(new Set(ids).size).toBe(8);
	});

	it('clamps page boundaries and represents an empty response without claiming a first result', () => {
		expect(getPagination(8, -2, 6).page).toBe(1);
		expect(getPagination(8, 99, 6).page).toBe(2);
		expect(getPagination(0, 2, 6)).toMatchObject({ page: 1, start: 0, end: 0 });
	});
});
