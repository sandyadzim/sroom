import { describe, expect, it } from 'vitest';
import { getPagination } from './pagination.ts';
import { demoCatalogCars } from '../data/demo-fleet.ts';

describe('catalogue pagination', () => {
	it('covers all eight distinct variants over pages of six without duplicates or omissions', () => {
		const first = getPagination(demoCatalogCars.length, 1, 6);
		const last = getPagination(demoCatalogCars.length, 2, 6);
		const ids = [
			...demoCatalogCars.slice(first.offset, first.end),
			...demoCatalogCars.slice(last.offset, last.end)
		].map((car) => car.id);
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
