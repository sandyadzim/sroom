import { describe, expect, it } from 'vitest';
import {
	catalogApiParams,
	catalogUrlParams,
	defaultCatalogFilters,
	readCatalogFilters
} from './catalog-filters.ts';

describe('Explore Cars query parameters', () => {
	it('maps all filters to the Catalog contract and round-trips through the page URL', () => {
		const filters = readCatalogFilters(
			new URLSearchParams({
				search: ' Brio RS ',
				brand: 'A916D894-F44B-4607-A26E-1132244F2031',
				category: 'city',
				seats: '05',
				transmission: 'automatic',
				price: '400-450',
				sort: 'price-asc'
			})
		);

		expect(Object.fromEntries(catalogApiParams(filters, 2, 6))).toEqual({
			search: 'Brio RS',
			brand_id: 'a916d894-f44b-4607-a26e-1132244f2031',
			category: 'city',
			seats: '5',
			transmission: 'automatic',
			min_price: '400001',
			max_price: '450000',
			sort: 'price-asc',
			page: '2',
			limit: '6'
		});
		const url = catalogUrlParams(filters, 2);
		expect(url.get('page')).toBe('2');
		expect(readCatalogFilters(url)).toEqual(filters);
	});

	it('ignores invalid filters and omits defaults from the page URL', () => {
		expect(
			readCatalogFilters(
				new URLSearchParams(
					'brand=Honda&category=truck&seats=-1&transmission=other&price=cheap&sort=old'
				)
			)
		).toEqual(defaultCatalogFilters);
		expect(catalogUrlParams(defaultCatalogFilters, 1).toString()).toBe('');
		expect(Object.fromEntries(catalogApiParams(defaultCatalogFilters, 1, 6))).toEqual({
			page: '1',
			limit: '6',
			sort: 'recommended'
		});
	});

	it('maps each price band without gaps or overlaps', () => {
		const ranges = [
			['under-300', null, '299999'],
			['300-350', '300000', '350000'],
			['350-400', '350001', '400000'],
			['400-450', '400001', '450000'],
			['450-500', '450001', '500000'],
			['over-500', '500001', null]
		] as const;

		for (const [price, min, max] of ranges) {
			const filters = readCatalogFilters(new URLSearchParams({ price }));
			const params = catalogApiParams(filters, 1, 6);
			expect([params.get('min_price'), params.get('max_price')]).toEqual([min, max]);
		}
	});

	it('keeps older price URLs usable', () => {
		const filters = readCatalogFilters(new URLSearchParams('price=over-700'));
		expect(catalogApiParams(filters, 1, 6).get('min_price')).toBe('700001');
	});
});
