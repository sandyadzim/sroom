import { afterEach, describe, expect, it, vi } from 'vitest';
import type { CatalogCarDetail } from '#lib/types/catalog.ts';
import { load } from './+page.server.ts';

vi.mock('$app/env/public', () => ({ PUBLIC_GATEWAY_BASE_URL: 'http://localhost:8080' }));
vi.mock('$app/paths', () => ({
	resolve: (_route: string, params: { slug: string }) => `/cars/${params.slug}`
}));

const cover = { url: 'https://images.example/cover.webp', alt_text: 'Front view', label: 'Cover' };
const rear = { url: 'https://images.example/rear.webp', alt_text: 'Rear view', label: 'Rear' };
const car: CatalogCarDetail = {
	id: 'variant-1',
	brand_id: 'brand-1',
	brand_name: 'Toyota',
	model: 'Innova Venturer',
	trim: '2.4 Diesel AT',
	slug: 'toyota-innova-venturer-2-4-diesel-at',
	status: 'active',
	category: 'mpv',
	seats: 7,
	transmission: 'automatic',
	body_type: 'mpv',
	fuel_type: 'diesel',
	daily_rate_idr: 780000,
	description: 'A seven-seat MPV.',
	cover,
	gallery: [cover, { url: null, alt_text: 'Interior', label: 'Interior' }, rear]
};
const event = { params: { slug: car.slug } } as Parameters<typeof load>[0];

function mockCatalog(detail: Response, related: Response) {
	const fetch = vi.fn().mockResolvedValueOnce(detail).mockResolvedValueOnce(related);
	vi.stubGlobal('fetch', fetch);
	return fetch;
}

afterEach(() => vi.unstubAllGlobals());

describe('public car detail loader', () => {
	it('loads the exact slug and ordered gallery and maps up to three related cards', async () => {
		const related = { ...car, id: 'variant-2', slug: 'another-innova', cover: null };
		const fetch = mockCatalog(Response.json({ data: car }), Response.json({ data: [related] }));

		expect(await load(event)).toMatchObject({
			car: {
				id: car.id,
				slug: car.slug,
				trim: car.trim,
				status: 'active',
				category: 'MPV',
				seats: 7,
				transmission: 'Automatic',
				bodyType: 'MPV',
				fuel: 'Diesel',
				description: car.description,
				dailyRate: 780000,
				gallery: [
					{ src: cover.url, alt: cover.alt_text, label: cover.label },
					{ src: rear.url, alt: rear.alt_text, label: rear.label }
				]
			},
			relatedCars: [{ id: related.id, href: '/cars/another-innova', image: null }],
			relatedUnavailable: false
		});
		const detailUrl = `http://localhost:8080/api/v1/catalog/cars/${car.slug}`;
		expect(fetch).toHaveBeenNthCalledWith(1, detailUrl, expect.anything());
		expect(fetch).toHaveBeenNthCalledWith(2, `${detailUrl}/related?limit=3`, expect.anything());
	});

	it('keeps an inactive banner car readable with no images and no related cars', async () => {
		mockCatalog(
			Response.json({ data: { ...car, status: 'inactive', cover: null, gallery: [] } }),
			Response.json({ data: [] })
		);
		expect(await load(event)).toMatchObject({
			car: { status: 'inactive', image: null, gallery: [] },
			relatedCars: [],
			relatedUnavailable: false
		});
	});

	it('keeps the detail available when related cars fail to load', async () => {
		mockCatalog(Response.json({ data: car }), new Response(null, { status: 502 }));
		expect(await load(event)).toMatchObject({
			car: { slug: car.slug },
			relatedCars: [],
			relatedUnavailable: true
		});
	});

	it.each([
		[404, 404],
		[502, 503]
	])('uses the recovery status for a %i detail response', async (apiStatus, pageStatus) => {
		mockCatalog(new Response(null, { status: apiStatus }), Response.json({ data: [] }));
		await expect(load(event)).rejects.toMatchObject({ status: pageStatus });
	});

	it('rejects invalid slugs before calling the gateway', async () => {
		const fetch = vi.fn();
		vi.stubGlobal('fetch', fetch);
		await expect(load({ ...event, params: { slug: '..' } })).rejects.toMatchObject({ status: 404 });
		expect(fetch).not.toHaveBeenCalled();
	});
});
