export type CatalogFilters = {
	search: string;
	brand: string;
	category: string;
	seats: string;
	transmission: string;
	price: string;
	sort: string;
};

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
	car_id: string;
	headline_line_1: string;
	headline_line_2: string;
	description: string;
	image_object_key: string;
};

export type BannerContent = Omit<BannerInput, 'car_id'> & {
	car_id: string | null;
	brand_id: string;
	brand_name: string;
	car_model: string;
	car_slug: string;
	image_url: string | null;
};

export type BannerSlot = {
	position: number;
	content: BannerContent | null;
};

export type SavedBanner = { data: BannerSlot; message: string };

export type PublicBanner = {
	slug: string;
	make: string;
	model: string;
	image: string | null;
	headline: [string, string];
	description: string;
};

export type BrandPage = {
	data: Brand[];
	page: { number: number; limit: number; total: number };
};

export type CarPhotoInput = {
	object_key: string;
	alt_text: string;
	label: string;
};

export type CatalogCarImage = Pick<CarPhotoInput, 'alt_text' | 'label'> & { url: string | null };

export type AdminCarImage = CatalogCarImage &
	CarPhotoInput & { id: string; kind: 'cover' | 'gallery' };

export type CatalogCarCard = {
	id: string;
	brand_id: string;
	brand_name: string;
	model: string;
	trim: string;
	slug: string;
	category: string;
	seats: number;
	transmission: string;
	daily_rate_idr: number;
	cover: CatalogCarImage | null;
};

export type CatalogCarDetail = CatalogCarCard & {
	status: 'active' | 'inactive';
	body_type: string;
	fuel_type: string;
	description: string;
	gallery: CatalogCarImage[];
};

export type AdminFeaturedCar = CatalogCarCard & { status: 'active' | 'inactive' };
export type PublicCatalogPage = {
	data: CatalogCarCard[];
	page: { number: number; limit: number; total: number };
};
export type PublicBrand = Pick<Brand, 'id' | 'name' | 'image_url'>;

export type AdminFeaturedSelection = AdminFeaturedCar & { position: number };
export type UpdateFeaturedCarsInput = { car_ids: (string | null)[] };
export type SavedFeaturedCars = { data: AdminFeaturedSelection[]; message: string };

export type AdminCar = AdminFeaturedCar & {
	created_at: string;
	updated_at: string;
};

export type AdminCarDetail = AdminCar & {
	body_type: string;
	fuel_type: string;
	description: string;
	images: AdminCarImage[];
};

export type CarFields = Pick<
	AdminCarDetail,
	| 'brand_id'
	| 'model'
	| 'trim'
	| 'category'
	| 'body_type'
	| 'seats'
	| 'transmission'
	| 'fuel_type'
	| 'description'
	| 'daily_rate_idr'
	| 'status'
>;

export type CreateCarInput = CarFields & { cover: CarPhotoInput; gallery: CarPhotoInput[] };
export type UpdateCarInput = Partial<CreateCarInput>;
export type SavedCar = { data: AdminCarDetail; message: string };

export type CarUnit = {
	id: string;
	car_id: string;
	internal_code: string;
	status: 'active' | 'inactive';
	created_at: string;
	updated_at: string;
};

export type CreateCarUnitInput = Pick<CarUnit, 'internal_code' | 'status'>;
export type UpdateCarUnitInput = Partial<CreateCarUnitInput>;
export type SavedCarUnit = { data: CarUnit; message: string };
