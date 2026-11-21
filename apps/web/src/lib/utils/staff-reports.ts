import type { ReportDay, ReportPeriod } from '#lib/types/admin-reports.ts';

export type ReportRangeErrors = {
	from?: string;
	to?: string;
	range?: string;
	query?: string;
};

export type ReportRangeState =
	| { kind: 'default' }
	| { kind: 'range'; period: ReportPeriod }
	| { kind: 'invalid'; errors: ReportRangeErrors; from: string; to: string };

const reportDatePattern = /^\d{4}-\d{2}-\d{2}$/;
const millisecondsPerDay = 24 * 60 * 60 * 1000;
const maximumReportDays = 366;

export function isReportDate(value: string) {
	if (!reportDatePattern.test(value)) return false;
	const date = new Date(`${value}T00:00:00Z`);
	return !Number.isNaN(date.getTime()) && date.toISOString().slice(0, 10) === value;
}

export function inclusiveReportDays(from: string, to: string) {
	const start = new Date(`${from}T00:00:00Z`).getTime();
	const end = new Date(`${to}T00:00:00Z`).getTime();
	return Math.round((end - start) / millisecondsPerDay) + 1;
}

export function validateReportRange(from: string, to: string): ReportRangeErrors {
	const errors: ReportRangeErrors = {};
	if (!isReportDate(from)) errors.from = 'Choose a valid start date.';
	if (!isReportDate(to)) errors.to = 'Choose a valid end date.';
	if (errors.from || errors.to) return errors;
	if (from > to) {
		errors.range = 'End date must be on or after the start date.';
	} else if (inclusiveReportDays(from, to) > maximumReportDays) {
		errors.range = 'Choose a range of 366 days or fewer.';
	}
	return errors;
}

export function parseReportRange(
	params: Pick<URLSearchParams, 'getAll' | 'keys'>
): ReportRangeState {
	const fromValues = params.getAll('from');
	const toValues = params.getAll('to');
	const hasUnknownKey = [...params.keys()].some((key) => key !== 'from' && key !== 'to');
	if (hasUnknownKey || fromValues.length > 1 || toValues.length > 1) {
		return {
			kind: 'invalid',
			errors: { query: 'Remove unsupported or duplicate date filters.' },
			from: fromValues[0] ?? '',
			to: toValues[0] ?? ''
		};
	}

	const from = fromValues[0] ?? '';
	const to = toValues[0] ?? '';
	if (!from && !to) return { kind: 'default' };
	if (!from || !to) {
		return {
			kind: 'invalid',
			errors: {
				from: from ? undefined : 'Choose both start and end dates.',
				to: to ? undefined : 'Choose both start and end dates.'
			},
			from,
			to
		};
	}

	const errors = validateReportRange(from, to);
	if (errors.from || errors.to || errors.range) return { kind: 'invalid', errors, from, to };
	return { kind: 'range', period: { from, to } };
}

export function reportPeriodFor(state: ReportRangeState) {
	return state.kind === 'range' ? state.period : undefined;
}

const percent = new Intl.NumberFormat('en-US', { style: 'percent', maximumFractionDigits: 1 });

export function formatConversion(ratio: number | null) {
	return ratio === null ? '—' : percent.format(ratio);
}

export function fleetDetail(counts: { active: number; inactive: number }) {
	return `${counts.active} active · ${counts.inactive} inactive`;
}

export function dailyChartData(daily: ReportDay[], value: 'bookings' | 'confirmed_value_idr') {
	return daily.map((row) => ({ day: row.date, value: row[value] }));
}
