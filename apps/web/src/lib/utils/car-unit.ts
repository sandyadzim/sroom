import type { CarUnit, CreateCarUnitInput, UpdateCarUnitInput } from '#lib/types/catalog.ts';

export function unitCodeError(value: string) {
	const code = value.trim();
	if (!code) return 'Enter an internal code.';
	if (Array.from(code).length > 100) return 'Use at most 100 characters.';
	if (/\p{Cc}/u.test(code)) return 'Remove control characters from the code.';
}

export function changedUnitFields(unit: CarUnit, input: CreateCarUnitInput): UpdateCarUnitInput {
	const changes: UpdateCarUnitInput = {};
	if (input.internal_code !== unit.internal_code) changes.internal_code = input.internal_code;
	if (input.status !== unit.status) changes.status = input.status;
	return changes;
}
