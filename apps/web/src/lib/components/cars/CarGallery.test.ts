import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import CarGallery from './CarGallery.svelte';

describe('CarGallery server rendering', () => {
	it('prioritizes the initial image and labels the enlarge control without requiring hydration', () => {
		const { body } = render(CarGallery, {
			props: {
				name: 'Toyota Fortuner',
				images: [{ src: '/fortuner.webp', alt: 'Illustrative Toyota Fortuner', label: 'Exterior' }]
			}
		});
		expect(body).toContain('fetchpriority="high"');
		expect(body).toContain('alt="Illustrative Toyota Fortuner"');
		expect(body).toContain('aria-label="Enlarge Toyota Fortuner image"');
		expect(body).toContain('Choose a vehicle image');
		expect(body).toContain('aria-label="Show exterior image"');
		expect(body).toContain('aria-pressed="true"');
	});

	it('keeps empty optional media usable and omits a nonfunctional enlarge control', () => {
		const { body } = render(CarGallery, { props: { name: 'Toyota Fortuner', images: [] } });
		expect(body).toContain('Image unavailable for Toyota Fortuner');
		expect(body).toContain('You can still explore the car details below.');
		expect(body).not.toContain('Enlarge Toyota Fortuner image');
	});
});
