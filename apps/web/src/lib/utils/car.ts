import type {
	AdminCarDetail,
	CarFields,
	CarPhotoInput,
	UpdateCarInput
} from '#lib/types/catalog.ts';

export const carCategories = [
	{ value: 'economy', label: 'Economy' },
	{ value: 'city', label: 'City' },
	{ value: 'mpv', label: 'MPV' },
	{ value: 'suv', label: 'SUV' },
	{ value: 'premium', label: 'Premium' }
] as const;

export const carBodyTypes = [
	{ value: 'hatchback', label: 'Hatchback' },
	{ value: 'sedan', label: 'Sedan' },
	{ value: 'mpv', label: 'MPV' },
	{ value: 'suv', label: 'SUV' }
] as const;

export const carTransmissions = [
	{ value: 'automatic', label: 'Automatic' },
	{ value: 'manual', label: 'Manual' }
] as const;

export const carFuelTypes = [
	{ value: 'petrol', label: 'Petrol' },
	{ value: 'diesel', label: 'Diesel' },
	{ value: 'hybrid', label: 'Hybrid' },
	{ value: 'electric', label: 'Electric' }
] as const;

export const carStatuses = [
	{ value: 'active', label: 'Active' },
	{ value: 'inactive', label: 'Inactive' }
] as const;

export function carVariantName(car: Pick<AdminCarDetail, 'brand_name' | 'model' | 'trim'>) {
	return [car.brand_name, car.model, car.trim].filter(Boolean).join(' ');
}

export function carOptionLabel(
	options: readonly { value: string; label: string }[],
	value: string
) {
	return options.find((option) => option.value === value)?.label ?? value;
}

export function carPhotoInput(photo: CarPhotoInput): CarPhotoInput {
	return {
		object_key: photo.object_key,
		alt_text: photo.alt_text.trim(),
		label: photo.label.trim()
	};
}

export function suggestCarPhotoMetadata(filename: string, vehicle: string, fallbackLabel: string) {
	const name = filename
		.replace(/\.[^.]+$/, '')
		.replace(/[._\s-]+/g, ' ')
		.trim();
	const isCameraName = /^(?:(?:img|image|dscf?|pxl|photo)\s*)?\d[\d\s]*$/i.test(name);
	const title = !name || isCameraName ? fallbackLabel : name[0].toUpperCase() + name.slice(1);
	const label = Array.from(title).slice(0, 100).join('').trimEnd();
	const alt =
		vehicle && !label.toLowerCase().includes(vehicle.toLowerCase())
			? `${vehicle} — ${label}`
			: label;
	return { label, alt_text: Array.from(alt).slice(0, 200).join('').trimEnd() };
}

export function changedCarFields(
	original: AdminCarDetail,
	fields: CarFields,
	cover: CarPhotoInput | undefined,
	gallery: CarPhotoInput[]
): UpdateCarInput {
	const changes: UpdateCarInput = {};
	for (const key of Object.keys(fields) as (keyof CarFields)[]) {
		if (fields[key] !== original[key]) Object.assign(changes, { [key]: fields[key] });
	}

	const originalCover = original.images.find((photo) => photo.kind === 'cover');
	const originalGallery = original.images.filter((photo) => photo.kind === 'gallery');
	if (cover && (!originalCover || !samePhoto(cover, originalCover))) changes.cover = cover;
	if (
		originalGallery.length !== gallery.length ||
		gallery.some((photo, index) => !samePhoto(photo, originalGallery[index]))
	) {
		changes.gallery = gallery;
	}
	return changes;
}

function samePhoto(photo: CarPhotoInput, other: CarPhotoInput) {
	return (
		photo.object_key === other.object_key &&
		photo.alt_text === other.alt_text &&
		photo.label === other.label
	);
}
