import { createRawSnippet } from 'svelte';
import { render } from 'svelte/server';
import { describe, expect, it } from 'vitest';
import NumberField from './NumberField.svelte';

const idr = createRawSnippet(() => ({ render: () => '<span>IDR</span>' }));
const percent = createRawSnippet(() => ({ render: () => '<span>%</span>' }));

describe('NumberField', () => {
	it('renders a grouped value with a raw form value and an accessible unit', () => {
		const { body } = render(NumberField, {
			props: { id: 'rate', name: 'rate', label: 'Daily rate', value: '650000', prepend: idr }
		});
		expect(body).toContain('for="rate"');
		expect(body).toContain('inputmode="numeric"');
		expect(body).toContain('value="650.000"');
		expect(body).toContain('type="hidden" name="rate" value="650000"');
		expect(body).toContain('aria-describedby="rate-prepend"');
	});

	it('keeps an empty value and associates the suffix and validation with the input', () => {
		const { body } = render(NumberField, {
			props: {
				id: 'percentage',
				name: 'percentage',
				label: 'Percentage',
				append: percent,
				error: 'Enter a percentage.',
				required: true
			}
		});
		expect(body).toContain('type="hidden" name="percentage" value=""');
		expect(body).toContain('aria-describedby="percentage-append percentage-error"');
		expect(body).toContain('aria-invalid="true"');
	});
});
