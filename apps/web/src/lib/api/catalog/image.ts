import { catalogAdminPath, staffApi } from '#lib/api/staff.ts';
import type { CatalogImageUpload } from '#lib/types/catalog.ts';

export async function uploadCatalogImage(file: File, signal?: AbortSignal) {
	const body = new FormData();
	body.append('file', file);
	const { data } = await staffApi.post<CatalogImageUpload>(`${catalogAdminPath}/images`, body, {
		signal
	});
	return data.data.object_key;
}
