import { catalogAdminPath, staffApi } from '#lib/api/staff.ts';
import type {
	AdminFeaturedSelection,
	SavedFeaturedCars,
	UpdateFeaturedCarsInput
} from '#lib/types/catalog.ts';

const featuredCarsEndpoint = `${catalogAdminPath}/featured-cars`;

export async function fetchFeaturedCars(signal?: AbortSignal) {
	const { data } = await staffApi.get<{ data: AdminFeaturedSelection[] }>(featuredCarsEndpoint, {
		signal
	});
	return data.data;
}

export async function updateFeaturedCars(input: UpdateFeaturedCarsInput) {
	const { data } = await staffApi.put<SavedFeaturedCars>(featuredCarsEndpoint, input);
	return data;
}
