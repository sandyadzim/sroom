import { carsEndpoint } from '#lib/api/catalog/car.ts';
import { catalogAdminPath, staffApi } from '#lib/api/staff.ts';
import type {
	CarUnit,
	CreateCarUnitInput,
	SavedCarUnit,
	UpdateCarUnitInput
} from '#lib/types/catalog.ts';

export function carUnitsEndpoint(carId: string) {
	return `${carsEndpoint}/${encodeURIComponent(carId)}/units`;
}

function unitPath(id: string) {
	return `${catalogAdminPath}/units/${encodeURIComponent(id)}`;
}

export async function fetchCarUnit(id: string, signal?: AbortSignal) {
	const { data } = await staffApi.get<{ data: CarUnit }>(unitPath(id), { signal });
	return data.data;
}

export async function createCarUnit(carId: string, input: CreateCarUnitInput) {
	const { data } = await staffApi.post<SavedCarUnit>(carUnitsEndpoint(carId), input);
	return data;
}

export async function updateCarUnit(id: string, input: UpdateCarUnitInput) {
	const { data } = await staffApi.patch<SavedCarUnit>(unitPath(id), input);
	return data;
}
