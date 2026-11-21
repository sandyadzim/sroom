import { describe, expect, it } from 'vitest';
import {
	dailyChartData,
	fleetDetail,
	formatConversion,
	inclusiveReportDays,
	isReportDate,
	parseReportRange,
	reportPeriodFor,
	validateReportRange
} from './staff-reports.ts';

describe('staff report date range', () => {
	it('uses the server default when both dates are absent', () => {
		expect(parseReportRange(new URLSearchParams())).toEqual({ kind: 'default' });
		expect(reportPeriodFor({ kind: 'default' })).toBeUndefined();
	});

	it('accepts a complete inclusive range', () => {
		const state = parseReportRange(new URLSearchParams('from=2026-11-15&to=2026-11-21'));
		expect(state).toEqual({
			kind: 'range',
			period: { from: '2026-11-15', to: '2026-11-21' }
		});
		expect(reportPeriodFor(state)).toEqual({ from: '2026-11-15', to: '2026-11-21' });
	});

	it('rejects partial, invalid, reversed, oversized, duplicate and unknown filters', () => {
		expect(parseReportRange(new URLSearchParams('from=2026-11-15'))).toMatchObject({
			kind: 'invalid',
			errors: { to: 'Choose both start and end dates.' }
		});
		expect(parseReportRange(new URLSearchParams('from=2026-11-31&to=2026-12-01'))).toMatchObject({
			kind: 'invalid',
			errors: { from: 'Choose a valid start date.' }
		});
		expect(parseReportRange(new URLSearchParams('from=2026-11-21&to=2026-11-15'))).toMatchObject({
			kind: 'invalid',
			errors: { range: 'End date must be on or after the start date.' }
		});
		expect(parseReportRange(new URLSearchParams('from=2026-01-01&to=2027-01-02'))).toMatchObject({
			kind: 'invalid',
			errors: { range: 'Choose a range of 366 days or fewer.' }
		});
		expect(
			parseReportRange(new URLSearchParams('from=2026-11-15&from=2026-11-16&to=2026-11-21'))
		).toMatchObject({ kind: 'invalid', errors: { query: expect.any(String) } });
		expect(
			parseReportRange(new URLSearchParams('from=2026-11-15&to=2026-11-21&page=2'))
		).toMatchObject({ kind: 'invalid', errors: { query: expect.any(String) } });
	});

	it('counts WITA calendar dates inclusively without timezone shifts', () => {
		expect(isReportDate('2026-11-21')).toBe(true);
		expect(isReportDate('2026-02-29')).toBe(false);
		expect(isReportDate('2024-02-29')).toBe(true);
		expect(inclusiveReportDays('2026-11-15', '2026-11-21')).toBe(7);
		expect(inclusiveReportDays('2024-01-01', '2024-12-31')).toBe(366);
		expect(validateReportRange('2026-11-21', '2026-11-21')).toEqual({});
	});
});

describe('staff report metric adapters', () => {
	it('formats conversion and keeps null conversion explicit', () => {
		expect(formatConversion(null)).toBe('—');
		expect(formatConversion(0.5)).toBe('50%');
		expect(formatConversion(0.4444)).toBe('44.4%');
	});

	it('formats fleet active and inactive counts', () => {
		expect(fleetDetail({ active: 23, inactive: 1 })).toBe('23 active · 1 inactive');
	});

	it('maps daily values in backend order and preserves zero-filled dates', () => {
		const daily = [
			{ date: '2026-11-15', bookings: 0, confirmed_value_idr: 0 },
			{ date: '2026-11-16', bookings: 3, confirmed_value_idr: 960_000 }
		];

		expect(dailyChartData(daily, 'bookings')).toEqual([
			{ day: '2026-11-15', value: 0 },
			{ day: '2026-11-16', value: 3 }
		]);
		expect(dailyChartData(daily, 'confirmed_value_idr')).toEqual([
			{ day: '2026-11-15', value: 0 },
			{ day: '2026-11-16', value: 960_000 }
		]);
	});
});
