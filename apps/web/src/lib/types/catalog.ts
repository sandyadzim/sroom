// UI request preparation only; not an approved backend transport contract.
export type CatalogFilters = {
	search: string;
	brand: string;
	seats: string;
	price: string;
	sort: string;
};

export const defaultCatalogFilters: CatalogFilters = {
	search: '',
	brand: '',
	seats: '',
	price: '',
	sort: 'recommended'
};

export function prepareCatalogRequest(filters: CatalogFilters, page: number, pageSize: number) {
	const prices: Record<string, { minPrice?: number; maxPrice?: number }> = {
		'under-400': { maxPrice: 400000 },
		'400-700': { minPrice: 400001, maxPrice: 700000 },
		'over-700': { minPrice: 700001 }
	};
	return {
		search: filters.search.trim() || undefined,
		brand: filters.brand || undefined,
		seats: filters.seats ? Number(filters.seats) : undefined,
		...prices[filters.price],
		sort: filters.sort,
		page,
		pageSize
	};
}

export type BrandStatus = 'active' | 'inactive';

export type Brand = {
	id: string;
	name: string;
	status: BrandStatus;
	image_object_key: string | null;
	image_filename: string | null;
	image_url: string | null;
	created_at: string;
	updated_at: string;
};

export type CreateBrandInput = {
	name: string;
	image_object_key?: string;
	image_filename?: string;
};

export type UpdateBrandInput = {
	name?: string;
	status?: BrandStatus;
	image_object_key?: string | null;
	image_filename?: string;
};

export type SavedBrand = { data: Brand; message: string };
export type CatalogImageUpload = { data: { object_key: string } };

export type BannerInput = {
	brand_id: string;
	car_model: string;
	car_slug: string;
	headline_line_1: string;
	headline_line_2: string;
	description: string;
	image_object_key: string;
};

export type BannerContent = BannerInput & {
	brand_name: string;
	image_url: string | null;
};

export type BannerSlot = {
	position: number;
	content: BannerContent | null;
};

export type SavedBanner = { data: BannerSlot; message: string };

export type BrandPage = {
	data: Brand[];
	page: { number: number; limit: number; total: number };
};
