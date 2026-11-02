import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import SelectField from './SelectField.svelte';

const options = [
	{ value: '', label: 'All brands' },
	{ value: 'Honda', label: 'Honda' },
	{ value: 'Toyota', label: 'Toyota' }
];

describe('SelectField server rendering', () => {
	it('shows an empty-valued option before hydration and keeps its form value', () => {
		const { body } = render(SelectField, {
			props: { id: 'brand', name: 'brand', label: 'Brand', options }
		});
		expect(body).toContain('All brands');
		expect(body).not.toContain('Select an option');
		expect(body).toContain('name="brand"');
		expect(body).toContain('value=""');
		expect(body).toContain('aria-labelledby="brand-label brand-value"');
	});

	it('shows a supplied selection and uses a placeholder only when no option matches', () => {
		const selected = render(SelectField, {
			props: { name: 'brand', label: 'Brand', options, value: 'Toyota' }
		});
		const empty = render(SelectField, {
			props: {
				name: 'brand',
				label: 'Brand',
				options: options.slice(1),
				placeholder: 'Choose a brand'
			}
		});
		expect(selected.body).toContain('Toyota');
		expect(selected.body).toContain('value="Toyota"');
		expect(empty.body).toContain('Choose a brand');
	});

	it('associates validation and help with the trigger and supports a disabled field', () => {
		const { body } = render(SelectField, {
			props: {
				id: 'brand',
				name: 'brand',
				label: 'Brand',
				options,
				required: true,
				disabled: true,
				help: 'Choose one brand.',
				error: 'A brand is required.'
			}
		});
		expect(body).toContain('aria-describedby="brand-help brand-error"');
		expect(body).toContain('aria-invalid="true"');
		expect(body).toContain('id="brand-error"');
		expect(body).toMatch(/<button[^>]*\bdisabled\b/);
	});
});
