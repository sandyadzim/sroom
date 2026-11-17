import { describe, expect, it } from 'vitest';
import { earliestPickupDate, inclusiveRentalDays, validateSchedule } from './schedule.ts';

describe('checkout schedule', () => {
	it('advances the earliest pickup at midnight in Samarinda', () => {
		expect(earliestPickupDate(new Date('2026-11-10T15:59:59Z'))).toBe('2026-11-11');
		expect(earliestPickupDate(new Date('2026-11-10T16:00:00Z'))).toBe('2026-11-12');
	});

	it('counts inclusive dates independently of pickup and return times', () => {
		expect(inclusiveRentalDays('2026-11-10', '2026-11-12')).toBe(3);
		expect(inclusiveRentalDays('2026-12-31', '2027-01-01')).toBe(2);
	});

	it('rejects dates before the earliest pickup and return before pickup', () => {
		const result = validateSchedule(
			{
				pickupDate: '2026-11-10',
				pickupTime: '09:00',
				returnDate: '2026-11-09',
				returnTime: '17:00'
			},
			'2026-11-11'
		);
		expect(result.pickupDate).toBeDefined();
		expect(result.returnDate).toBeDefined();
	});

	it('rejects a return at or before the pickup time on the same day', () => {
		const result = validateSchedule(
			{
				pickupDate: '2026-11-12',
				pickupTime: '09:00',
				returnDate: '2026-11-12',
				returnTime: '09:00'
			},
			'2026-11-11'
		);
		expect(result.returnTime).toBeDefined();
	});
});
