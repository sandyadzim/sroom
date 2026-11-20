import { error } from '@sveltejs/kit';
import { fetchPublicCar } from '#lib/api/catalog/public.server.ts';
import type { PageServerLoad } from './$types';
export const load = (async ({ params }) => {
	if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(params.slug)) error(404, 'This car could not be found.');
	let car;
	try {
		car = await fetchPublicCar(params.slug);
	} catch {
		error(503, 'This car could not be loaded. Please try again.');
	}
	if (!car) error(404, 'This car could not be found.');
	if (car.status !== 'active') error(404, 'This car is currently unavailable for booking.');
	return { car };
}) satisfies PageServerLoad;
