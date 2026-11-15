import { describe, expect, it } from 'vitest';
import type { AdminCarDetail, CarFields, CarPhotoInput } from '#lib/types/catalog.ts';
import { carPhotoInput, changedCarFields, suggestCarPhotoMetadata } from './car';

describe('suggestCarPhotoMetadata', () => {
	it('uses a readable filename and adds the vehicle name to the alt text', () => {
		expect(suggestCarPhotoMetadata('rear_view.webp', 'Toyota Avanza', 'Gallery photo 1')).toEqual({
			label: 'Rear view',
			alt_text: 'Toyota Avanza — Rear view'
		});
	});

	it('uses the photo role for generic camera filenames', () => {
		expect(suggestCarPhotoMetadata('IMG_3048.JPG', 'Toyota Avanza', 'Cover photo')).toEqual({
			label: 'Cover photo',
			alt_text: 'Toyota Avanza — Cover photo'
		});
	});

	it('avoids repeating the vehicle name and keeps suggestions within API limits', () => {
		expect(
			suggestCarPhotoMetadata('toyota_avanza-front.png', 'Toyota Avanza', 'Cover photo')
		).toEqual({ label: 'Toyota avanza front', alt_text: 'Toyota avanza front' });
		const suggestion = suggestCarPhotoMetadata(
			`${'é'.repeat(110)}.png`,
			'Avanza '.repeat(16).trim(),
			'Cover photo'
		);
		expect(Array.from(suggestion.label)).toHaveLength(100);
		expect(Array.from(suggestion.alt_text)).toHaveLength(200);
	});
});

const fields: CarFields = {
	brand_id: 'brand',
	model: 'Fortuner',
	trim: 'VRZ',
	category: 'suv',
	body_type: 'suv',
	seats: 7,
	transmission: 'automatic',
	fuel_type: 'diesel',
	description: 'A seven-seat SUV.',
	daily_rate_idr: 850000,
	status: 'inactive'
};
const cover = {
	object_key: 'cover.webp',
	alt_text: 'Front view of a white Fortuner',
	label: 'Front'
};
const gallery = [
	{ object_key: 'rear.webp', alt_text: 'Rear view of a white Fortuner', label: 'Rear' },
	{ object_key: 'dash.webp', alt_text: 'Dashboard of a Fortuner', label: 'Dashboard' }
];
const original: AdminCarDetail = {
	...fields,
	id: 'car',
	brand_name: 'Toyota',
	slug: 'toyota-fortuner-vrz',
	cover: { ...cover, url: 'https://example.test/cover.webp' },
	images: [
		{ ...cover, id: 'cover', kind: 'cover', url: null },
		...gallery.map((photo) => ({
			...photo,
			id: photo.object_key,
			kind: 'gallery' as const,
			url: null
		}))
	],
	created_at: '2026-11-15T08:00:00Z',
	updated_at: '2026-11-15T08:00:00Z'
};

describe('changedCarFields', () => {
	it('omits unchanged fields and photo references', () => {
		expect(changedCarFields(original, fields, cover, gallery)).toEqual({});
	});

	it('changes the empty trim and status without resending image references or slug', () => {
		expect(
			changedCarFields(original, { ...fields, trim: '', status: 'active' }, cover, gallery)
		).toEqual({ trim: '', status: 'active' });
	});

	it('sends only the edited cover and the replacement gallery, without read-only photo fields', () => {
		const nextCover = carPhotoInput({ ...original.images[0], label: ' Front exterior ' });
		const nextGallery: CarPhotoInput[] = [gallery[1]];
		expect(changedCarFields(original, fields, nextCover, nextGallery)).toEqual({
			cover: { ...cover, label: 'Front exterior' },
			gallery: nextGallery
		});
	});
});
