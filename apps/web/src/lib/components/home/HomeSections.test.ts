import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import type { PublicBanner } from '#lib/types/catalog.ts';
import Hero from './Hero.svelte';
import FeaturedFleet from './FeaturedFleet.svelte';

const banner: PublicBanner = {
	slug: 'honda-brio-rs',
	make: 'Honda',
	model: 'Brio',
	image: null,
	headline: ['Small car.', 'Big city energy.'],
	description: 'A compact hatchback for everyday drives.'
};

describe('homepage sections', () => {
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
