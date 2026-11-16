import { describe, expect, it } from 'vitest';
import type { CarUnit } from '#lib/types/catalog.ts';
import { changedUnitFields, unitCodeError } from './car-unit';

const unit: CarUnit = {
	id: 'unit-1',
	car_id: 'car-1',
	internal_code: 'INNOVA-001',
	status: 'inactive',
	created_at: '2026-11-16T00:00:00Z',
	updated_at: '2026-11-16T00:00:00Z'
};

describe('unitCodeError', () => {
	it('requires a code and rejects embedded control characters', () => {
		expect(unitCodeError('  ')).toBe('Enter an internal code.');
		expect(unitCodeError('INNOVA\t001')).toBe('Remove control characters from the code.');
	});

	it('checks the trimmed code by Unicode character count', () => {
		expect(unitCodeError(`  ${'🚗'.repeat(100)}  `)).toBeUndefined();
		expect(unitCodeError('🚗'.repeat(101))).toBe('Use at most 100 characters.');
	});
});

describe('changedUnitFields', () => {
	it('sends only the changed code and status', () => {
		expect(
			changedUnitFields(unit, { internal_code: unit.internal_code, status: 'active' })
		).toEqual({
			status: 'active'
		});
		expect(changedUnitFields(unit, { internal_code: 'INNOVA-002', status: unit.status })).toEqual({
			internal_code: 'INNOVA-002'
		});
		expect(changedUnitFields(unit, unit)).toEqual({});
	});
});
