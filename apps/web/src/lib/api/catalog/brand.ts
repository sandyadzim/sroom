import { staffApi, staffApiPath } from '#lib/api/staff.ts';
import type {
	CatalogImageUpload,
	CreateBrandInput,
	SavedBrand,
	UpdateBrandInput
} from '#lib/types/catalog.ts';

const catalogAdminPath = `${staffApiPath}/catalog/admin`;
export const brandsEndpoint = `${catalogAdminPath}/brands`;

export async function createBrand(input: CreateBrandInput) {
	const { data } = await staffApi.post<SavedBrand>(brandsEndpoint, input);
	return data;
}

export async function updateBrand(id: string, input: UpdateBrandInput) {
	const { data } = await staffApi.patch<SavedBrand>(
		`${brandsEndpoint}/${encodeURIComponent(id)}`,
		input
	);
	return data;
}

export async function uploadCatalogImage(file: File, signal?: AbortSignal) {
	const body = new FormData();
	body.append('file', file);
	const { data } = await staffApi.post<CatalogImageUpload>(`${catalogAdminPath}/images`, body, {
		signal
	});
	return data.data.object_key;
}
