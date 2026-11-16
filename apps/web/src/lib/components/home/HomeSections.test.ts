import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { PublicBanner } from '#lib/types/catalog.ts';
import Hero from './Hero.svelte';
import FeaturedFleet from './FeaturedFleet.svelte';
import BrowseBrands from './BrowseBrands.svelte';

const banner: PublicBanner = {
	slug: 'honda-brio-rs',
	make: 'Honda',
	model: 'Brio',
	image: null,
	headline: ['Small car.', 'Big city energy.'],
	description: 'A compact hatchback for everyday drives.'
};

describe('homepage sections', () => {
	it('links brand logos to Explore Cars and limits the strip to five images', () => {
		const hondaId = 'a916d894-f44b-4607-a26e-1132244f2031';
		const suzukiId = 'b96ea552-f060-4fef-aedf-624bd1baec8f';
		const { body } = render(BrowseBrands, {
			props: {
				brands: [
					{ id: hondaId, name: 'Honda', image_url: '/honda.png' },
					{ id: suzukiId, name: 'Suzuki', image_url: null },
					...Array.from({ length: 5 }, (_, index) => ({
						id: `brand-${index}`,
						name: `Brand ${index}`,
						image_url: `/brand-${index}.png`
					}))
				]
			}
		});

		expect(body).toContain(`href="/cars?brand=${hondaId}"`);
		expect(body).not.toContain(`href="/cars?brand=${suzukiId}"`);
		expect(body).toContain('src="/honda.png"');
		expect(body).toContain('aria-label="Explore Honda cars"');
		expect(body).not.toContain('>Honda<');
		expect(body).not.toContain('src="/brand-4.png"');
		expect(body.match(/<img /g)).toHaveLength(5);
		expect(render(BrowseBrands, { props: { brands: [] } }).body).not.toContain('id="brands"');
	});

	it('links a single banner to its Catalog slug without thumbnails or a missing image tag', () => {
		const { body } = render(Hero, { props: { banners: [banner] } });

		expect(body).toContain('href="/cars/honda-brio-rs"');
		expect(body).toContain('Book Now — Honda Brio');
		expect(body).not.toContain('aria-pressed');
		expect(body).not.toContain('<img');
	});

	it('keeps Explore Cars available with no banners', () => {
		const { body } = render(Hero, { props: { banners: [] } });

		expect(body).toContain('href="/cars"');
		expect(body).toContain('Explore Cars');
	});

	it('distinguishes an empty featured selection from an unavailable feed', () => {
		const empty = render(FeaturedFleet, { props: { cars: [], unavailable: false } }).body;
		const unavailable = render(FeaturedFleet, { props: { cars: [], unavailable: true } }).body;

		expect(empty).toContain('There are no featured cars right now.');
		expect(unavailable).toContain('Featured cars are temporarily unavailable.');
		expect(empty).toContain('View all cars');
		expect(unavailable).toContain('View all cars');
	});
});
