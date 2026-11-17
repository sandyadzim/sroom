import { PUBLIC_GATEWAY_BASE_URL } from '$app/env/public';
import type {
	CatalogCarCard,
	CatalogCarDetail,
	PublicBanner,
	PublicBrand,
	PublicCatalogPage
} from '#lib/types/catalog.ts';

class CatalogRequestError extends Error {
	constructor(readonly status: number) {
		super(`Catalog request failed (${status})`);
	}
}

async function fetchCatalog<T>(endpoint: string): Promise<T> {
	const baseUrl = PUBLIC_GATEWAY_BASE_URL || 'http://localhost:8080';
	const response = await fetch(`${baseUrl.replace(/\/+$/, '')}/api/v1/catalog/${endpoint}`, {
		signal: AbortSignal.timeout(5_000)
	});
	if (!response.ok) throw new CatalogRequestError(response.status);
	return response.json();
}

async function fetchItems<T>(endpoint: string): Promise<T[]> {
	const { data } = await fetchCatalog<{ data: T[] }>(endpoint);
	if (!Array.isArray(data)) throw new Error('Invalid Catalog list response');
	return data;
}

export function fetchHomeBanners() {
	return fetchItems<PublicBanner>('home/banners');
}

export function fetchHomeFeaturedCars() {
	return fetchItems<CatalogCarCard>('home/featured-cars');
}

export function fetchPublicBrands(activeOnly = false) {
	return fetchItems<PublicBrand>(activeOnly ? 'brands?status=active' : 'brands');
}

export async function fetchPublicCar(slug: string) {
	try {
		const { data } = await fetchCatalog<{ data: CatalogCarDetail }>(
			`cars/${encodeURIComponent(slug)}`
		);
		if (!Array.isArray(data?.gallery)) throw new Error('Invalid Catalog car detail response');
		return data;
	} catch (error) {
		if (error instanceof CatalogRequestError && error.status === 404) return null;
		throw error;
	}
}

export function fetchRelatedCars(slug: string) {
	return fetchItems<CatalogCarCard>(`cars/${encodeURIComponent(slug)}/related?limit=3`);
}

export async function fetchPublicCars(params: URLSearchParams) {
	const result = await fetchCatalog<PublicCatalogPage>(`cars?${params}`);
	if (
		!Array.isArray(result.data) ||
		!Number.isSafeInteger(result.page?.total) ||
		result.page.total < 0
	) {
		throw new Error('Invalid Catalog page response');
	}
	return result;
}
