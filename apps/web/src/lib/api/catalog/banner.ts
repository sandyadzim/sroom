import { catalogAdminPath, staffApi } from '#lib/api/staff.ts';
import type { BannerInput, BannerSlot, SavedBanner } from '#lib/types/catalog.ts';

const bannersEndpoint = `${catalogAdminPath}/banners`;

export async function fetchBannerSlots(signal?: AbortSignal) {
	const { data } = await staffApi.get<{ data: BannerSlot[] }>(bannersEndpoint, { signal });
	return data.data;
}

export async function fetchBannerSlot(position: number, signal?: AbortSignal) {
	const { data } = await staffApi.get<{ data: BannerSlot }>(`${bannersEndpoint}/${position}`, {
		signal
	});
	return data.data;
}

export async function saveBannerSlot(position: number, input: BannerInput) {
	const { data } = await staffApi.put<SavedBanner>(`${bannersEndpoint}/${position}`, input);
	return data;
}

export async function clearBannerSlot(position: number) {
	const { data } = await staffApi.delete<SavedBanner>(`${bannersEndpoint}/${position}`);
	return data;
}
