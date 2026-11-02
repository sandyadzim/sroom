// Public card view model, not a backend API or physical-unit contract.
export type CarPreview = {
	id: string;
	brand: string;
	model: string;
	trim: string;
	seats: number;
	transmission: string;
	dailyRate: number;
	image: string;
	href?: string;
};

export type CarGalleryImage = {
	src: string;
	alt: string;
	label: string;
};

// Frontend detail view model, not a catalogue API or quote contract.
export type CarDetail = CarPreview & {
	category: string;
	fuel: string;
	description: string;
	gallery: CarGalleryImage[];
};
