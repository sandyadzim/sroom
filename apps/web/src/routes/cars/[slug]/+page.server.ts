import { error } from '@sveltejs/kit';
import { fetchPublicCar, fetchRelatedCars } from '#lib/api/catalog/public.server.ts';
import { toCarDetail, toCarPreview } from '#lib/utils/car-preview.ts';
import type { PageServerLoad } from './$types';

export const load = (async ({ params }) => {
	if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(params.slug)) {
		error(404, 'This car could not be found.');
	}

	const [detail, related] = await Promise.allSettled([
		fetchPublicCar(params.slug),
		fetchRelatedCars(params.slug)
	]);

	if (detail.status === 'rejected') error(503, 'This car could not be loaded.');
	if (!detail.value) error(404, 'This car could not be found.');

	return {
		car: toCarDetail(detail.value),
		relatedCars: related.status === 'fulfilled' ? related.value.map(toCarPreview) : [],
		relatedUnavailable: related.status === 'rejected'
	};
}) satisfies PageServerLoad;
