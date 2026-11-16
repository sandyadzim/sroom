import { catalogAdminPath, staffApi } from '#lib/api/staff.ts';
import type {
	AdminCar,
	AdminCarDetail,
	CreateCarInput,
	SavedCar,
	UpdateCarInput
} from '#lib/types/catalog.ts';
import type { TablePage } from '#lib/types/data-table.ts';

export const carsEndpoint = `${catalogAdminPath}/cars`;

function carPath(id: string) {
	return `${carsEndpoint}/${encodeURIComponent(id)}`;
}

export async function fetchActiveCarsForSelect(signal?: AbortSignal) {
	const cars: AdminCar[] = [];
	const limit = 100;
	for (let page = 1; ; page += 1) {
		const { data } = await staffApi.get<TablePage<AdminCar>>(carsEndpoint, {
			params: { page, limit, sort: 'name-asc', status: 'active' },
			signal
		});
		cars.push(...data.data);
		if (!data.data.length || cars.length >= data.page.total) return cars;
	}
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
