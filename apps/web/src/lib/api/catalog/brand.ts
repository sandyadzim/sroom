import { catalogAdminPath, staffApi } from '#lib/api/staff.ts';
import type {
	Brand,
	BrandPage,
	CreateBrandInput,
	SavedBrand,
	UpdateBrandInput
} from '#lib/types/catalog.ts';

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

export async function fetchBrandsForSelect(signal?: AbortSignal) {
	const brands: Brand[] = [];
	const limit = 100;
	for (let page = 1; ; page += 1) {
		const { data } = await staffApi.get<BrandPage>(brandsEndpoint, {
			params: { page, limit },
			signal
		});
		brands.push(...data.data);
		if (data.data.length < limit || page * limit >= data.page.total) return brands;
	}
}
