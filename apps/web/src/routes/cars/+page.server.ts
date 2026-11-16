import { redirect } from '@sveltejs/kit';
import { fetchPublicBrands, fetchPublicCars } from '#lib/api/catalog/public.server.ts';
import { toCarPreview } from '#lib/utils/car-preview.ts';
import {
	catalogApiParams,
	catalogUrlParams,
	readCatalogFilters
} from '#lib/utils/catalog-filters.ts';
import { getPagination } from '#lib/utils/pagination.ts';
import type { PageServerLoad } from './$types';

export const load = (async ({ url }) => {
	const filters = readCatalogFilters(url.searchParams);
	const page = Number(url.searchParams.get('page'));
	const requestedPage = Number.isSafeInteger(page) && page > 0 ? page : 1;
	const pageSize = 6;
	const [cars, brands] = await Promise.allSettled([
		fetchPublicCars(catalogApiParams(filters, requestedPage, pageSize)),
		fetchPublicBrands()
	]);

	const total = cars.status === 'fulfilled' ? cars.value.page.total : 0;
	const pagination = getPagination(total, requestedPage, pageSize);
	if (cars.status === 'fulfilled' && requestedPage !== pagination.page) {
		const params = catalogUrlParams(filters, pagination.page);
		redirect(307, params.size ? `${url.pathname}?${params}` : url.pathname);
	}

	return {
		filters,
		cars: cars.status === 'fulfilled' ? cars.value.data.map(toCarPreview) : [],
		brands: brands.status === 'fulfilled' ? brands.value : [],
		total,
		pagination,
		unavailable: cars.status === 'rejected',
		brandsUnavailable: brands.status === 'rejected'
	};
}) satisfies PageServerLoad;
