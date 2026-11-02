import { error } from '@sveltejs/kit';
import { getCarDetail, getRelatedCars } from '#lib/data/car-details.ts';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ params }) => {
	const car = getCarDetail(params.slug);
	if (!car) error(404, 'This car could not be found.');
	return { car, relatedCars: getRelatedCars(car.id) };
};
