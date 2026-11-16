import { PUBLIC_GATEWAY_BASE_URL } from '$app/env/public';
import type { CatalogCarCard, PublicBanner } from '#lib/types/catalog.ts';

async function fetchHomeItems<T>(endpoint: 'banners' | 'featured-cars'): Promise<T[]> {
	const baseUrl = PUBLIC_GATEWAY_BASE_URL || 'http://localhost:8080';
	const response = await fetch(`${baseUrl.replace(/\/+$/, '')}/api/v1/catalog/home/${endpoint}`, {
		signal: AbortSignal.timeout(5_000)
	});
	if (!response.ok) throw new Error(`Could not load ${endpoint} (${response.status})`);

	const { data }: { data: T[] } = await response.json();
	if (!Array.isArray(data)) throw new Error(`Invalid ${endpoint} response`);
	return data;
}

export function fetchHomeBanners() {
	return fetchHomeItems<PublicBanner>('banners');
}

export function fetchHomeFeaturedCars() {
	return fetchHomeItems<CatalogCarCard>('featured-cars');
}
