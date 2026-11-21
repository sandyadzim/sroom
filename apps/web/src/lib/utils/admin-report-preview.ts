import { adminReportFixtures, previewFleetCounts } from './admin-report-preview-data.ts';
import type { BookingStatus } from '#lib/types/admin-transactions.ts';

export const reportPreviewPeriod = { from: '2026-11-10', to: '2026-11-17' };

export function isReportDate(value: string) {
	if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
	const date = new Date(`${value}T00:00:00Z`);
	return !Number.isNaN(date.getTime()) && date.toISOString().slice(0, 10) === value;
}

export function getReportPeriod(params: Pick<URLSearchParams, 'get'>) {
	const from = params.get('from') ?? reportPreviewPeriod.from;
	const to = params.get('to') ?? reportPreviewPeriod.to;
	return isReportDate(from) && isReportDate(to) && from <= to ? { from, to } : reportPreviewPeriod;
}

export function getAdminReportPreview({ from, to } = reportPreviewPeriod) {
	const statusCounts: Record<BookingStatus, number> = {
		pending_payment: 0,
		confirmed: 0,
		expired: 0,
		payment_review: 0
	};
	const byDay = new Map<string, { day: string; bookings: number; value: number }>();
	const byModel = new Map<string, { model: string; bookings: number }>();
	const recent = [];
	let confirmedValue = 0;
	let confirmedPaid = 0;
	let pendingPayments = 0;
	let total = 0;

	for (const { booking, payment } of adminReportFixtures) {
		// Fixture timestamps carry the WITA offset.
		const day = booking.createdAt.slice(0, 10);
		if (day < from || day > to) continue;

		total += 1;
		statusCounts[booking.status] += 1;
		if (recent.length < 5) recent.push(booking);
		if (payment?.status === 'pending') pendingPayments += 1;

		const daily = byDay.get(day) ?? { day, bookings: 0, value: 0 };
		daily.bookings += 1;
		if (booking.status === 'confirmed' && payment?.status === 'paid') {
			confirmedPaid += 1;
			confirmedValue += booking.total;
			daily.value += booking.total;
		}
		byDay.set(day, daily);

		const model = byModel.get(booking.model) ?? { model: booking.model, bookings: 0 };
		model.bookings += 1;
		byModel.set(booking.model, model);
	}

	return {
		total,
		confirmedValue,
		pendingPayments,
		conversion: total ? confirmedPaid / total : null,
		statusCounts,
		fleet: previewFleetCounts,
		recent,
		days: [...byDay.values()].sort((a, b) => a.day.localeCompare(b.day)),
		models: [...byModel.values()].sort(
			(a, b) => b.bookings - a.bookings || a.model.localeCompare(b.model)
		)
	};
}
