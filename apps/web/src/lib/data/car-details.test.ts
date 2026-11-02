import { describe, expect, it } from 'vitest';
import { getCarDetail, getRelatedCars } from './car-details.ts';
import { demoCatalogCars } from './demo-fleet.ts';
import { load } from '../../routes/cars/[slug]/+page.ts';

describe('public car detail fixtures', () => {
	it('resolves every catalogue variant without changing its displayed price or specifications', () => {
		for (const preview of demoCatalogCars) {
			const detail = getCarDetail(preview.id);
			expect(detail).toMatchObject(preview);
			expect(detail?.gallery[0].src).toBe(preview.image);
			expect(detail?.fuel).toBe(preview.model === 'Fortuner' ? 'Diesel' : 'Petrol');
			expect(detail?.description).toBeTruthy();
		}
	});

	it('returns no detail for an unknown slug, including inherited property names', () => {
		expect(getCarDetail('not-published')).toBeUndefined();
		expect(getCarDetail('constructor')).toBeUndefined();
	});

	it('excludes the selected variant from related cards', () => {
		for (const car of demoCatalogCars) {
			const related = getRelatedCars(car.id);
			expect(related).toHaveLength(3);
			expect(related.some((item) => item.id === car.id)).toBe(false);
			expect(new Set(related.map((item) => item.id)).size).toBe(3);
			expect(new Set(related.map((item) => item.model)).size).toBe(3);
			expect(related.some((item) => item.model === car.model)).toBe(false);
		}
	});

	it('loads the exact public variant and rejects a missing route with HTTP 404', async () => {
		const event = { params: { slug: 'brio-satya-e-cvt' } } as Parameters<typeof load>[0];
		const data = await load(event);
		expect(data?.car.trim).toBe('Satya E CVT');
		expect(data?.car.dailyRate).toBe(275000);
		expect(() => load({ ...event, params: { slug: 'missing' } })).toThrow(
			expect.objectContaining({ status: 404 })
		);
	});
});
