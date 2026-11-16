import type { CatalogFilters } from '#lib/types/catalog.ts';
import { carCategories, carTransmissions } from './car.ts';

export const emptyCatalogFilters: Omit<CatalogFilters, 'sort'> = {
	search: '',
	brand: '',
	category: '',
	seats: '',
	transmission: '',
	price: ''
};

export const defaultCatalogFilters: CatalogFilters = {
	...emptyCatalogFilters,
	sort: 'recommended'
};

export const catalogSorts = [
	{ value: 'recommended', label: 'Recommended' },
	{ value: 'price-asc', label: 'Price: low to high' },
	{ value: 'price-desc', label: 'Price: high to low' },
	{ value: 'name-asc', label: 'Name: A to Z' }
];

export const catalogPrices: {
	value: string;
	label: string;
	params: Record<string, string>;
	legacy?: boolean;
}[] = [
	{ value: 'under-300', label: 'Below IDR 300.000', params: { max_price: '299999' } },
	{
		value: '300-350',
		label: 'IDR 300.000–350.000',
		params: { min_price: '300000', max_price: '350000' }
	},
	{
		value: '350-400',
		label: 'IDR 350.000–400.000',
		params: { min_price: '350001', max_price: '400000' }
	},
	{
		value: '400-450',
		label: 'IDR 400.000–450.000',
		params: { min_price: '400001', max_price: '450000' }
	},
	{
		value: '450-500',
		label: 'IDR 450.000–500.000',
		params: { min_price: '450001', max_price: '500000' }
	},
	{ value: 'over-500', label: 'Above IDR 500.000', params: { min_price: '500001' } },
	{ value: 'under-400', label: 'Up to IDR 400.000', params: { max_price: '400000' }, legacy: true },
	{
		value: '400-700',
		label: 'IDR 400.001–700.000',
		params: { min_price: '400001', max_price: '700000' },
		legacy: true
	},
	{ value: 'over-700', label: 'Above IDR 700.000', params: { min_price: '700001' }, legacy: true }
];

export function readCatalogFilters(params: URLSearchParams): CatalogFilters {
	const brand = params.get('brand') ?? '';
	const seats = params.get('seats') ?? '';

	function readOption(key: keyof CatalogFilters, options: readonly { value: string }[]) {
		return (
			options.find(({ value }) => value === params.get(key))?.value ?? defaultCatalogFilters[key]
		);
	}

	return {
		search: params.get('search')?.replaceAll('\0', '').trim() ?? '',
		brand: /^[a-f\d]{8}(?:-[a-f\d]{4}){3}-[a-f\d]{12}$/i.test(brand) ? brand.toLowerCase() : '',
		category: readOption('category', carCategories),
		seats:
			/^\d+$/.test(seats) && Number(seats) > 0 && Number(seats) <= 2147483647
				? String(Number(seats))
				: '',
		transmission: readOption('transmission', carTransmissions),
		price: readOption('price', catalogPrices),
		sort: readOption('sort', catalogSorts)
	};
}

export function catalogUrlParams(filters: CatalogFilters, page: number) {
	const params = new URLSearchParams();
	for (const [key, value] of Object.entries(filters)) {
		if (value && !(key === 'sort' && value === defaultCatalogFilters.sort)) params.set(key, value);
	}
	if (page > 1) params.set('page', String(page));
	return params;
}

export function catalogApiParams(filters: CatalogFilters, page: number, limit: number) {
	const params = new URLSearchParams({
		page: String(page),
		limit: String(limit),
		sort: filters.sort
	});
	const price = catalogPrices.find(({ value }) => value === filters.price);
	for (const [key, value] of Object.entries(price?.params ?? {})) params.set(key, value);
	if (filters.search) params.set('search', filters.search);
	if (filters.brand) params.set('brand_id', filters.brand);
	if (filters.category) params.set('category', filters.category);
	if (filters.seats) params.set('seats', filters.seats);
	if (filters.transmission) params.set('transmission', filters.transmission);
	return params;
}
