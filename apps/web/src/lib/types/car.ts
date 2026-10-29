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
