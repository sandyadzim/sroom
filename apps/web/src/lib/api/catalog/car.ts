import { catalogAdminPath, staffApi } from '#lib/api/staff.ts';
import type {
	AdminCarDetail,
	CreateCarInput,
	SavedCar,
	UpdateCarInput
} from '#lib/types/catalog.ts';

export const carsEndpoint = `${catalogAdminPath}/cars`;

function carPath(id: string) {
	return `${carsEndpoint}/${encodeURIComponent(id)}`;
}

export async function fetchCar(id: string, signal?: AbortSignal) {
	const { data } = await staffApi.get<{ data: AdminCarDetail }>(carPath(id), { signal });
	return data.data;
}

export async function createCar(input: CreateCarInput) {
	const { data } = await staffApi.post<SavedCar>(carsEndpoint, input);
	return data;
}

export async function updateCar(id: string, input: UpdateCarInput) {
	const { data } = await staffApi.patch<SavedCar>(carPath(id), input);
	return data;
}
