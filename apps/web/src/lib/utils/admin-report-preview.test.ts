import { describe, expect, it } from 'vitest';
import {
	getAdminReportPreview,
	getReportPeriod,
	reportPreviewPeriod
} from './admin-report-preview.ts';

describe('admin report preview', () => {
	it('includes both date boundaries and excludes paid bookings under review from confirmed value', () => {
		const report = getAdminReportPreview({ from: '2026-11-17', to: '2026-11-17' });

		expect(report.total).toBe(3);
		expect(report.confirmedValue).toBe(850_000);
		expect(report.conversion).toBe(1 / 3);
		expect(report.pendingPayments).toBe(1);
		expect(report.statusCounts).toEqual({
			confirmed: 1,
			pending_payment: 1,
			payment_review: 1,
			expired: 0
		});
		expect(report.days).toEqual([{ day: '2026-11-17', bookings: 3, value: 850_000 }]);
	});

	it('uses the selected period for trend, status, and model totals', () => {
		const report = getAdminReportPreview();

		expect(report.total).toBe(24);
		expect(report.days.reduce((total, row) => total + row.bookings, 0)).toBe(report.total);
		expect(report.days.reduce((total, row) => total + row.value, 0)).toBe(report.confirmedValue);
		expect(report.models.reduce((total, row) => total + row.bookings, 0)).toBe(report.total);
		expect(Object.values(report.statusCounts).reduce((total, count) => total + count, 0)).toBe(
			report.total
		);
	});

	it('has no conversion rate when there are no bookings', () => {
		const report = getAdminReportPreview({ from: '2026-11-18', to: '2026-11-20' });

		expect(report.total).toBe(0);
		expect(report.confirmedValue).toBe(0);
		expect(report.conversion).toBeNull();
		expect(report.days).toEqual([]);
		expect(report.models).toEqual([]);
	});

	it('restores valid URL dates and falls back for an invalid or reversed range', () => {
		expect(getReportPeriod(new URLSearchParams('from=2026-11-12&to=2026-11-14'))).toEqual({
			from: '2026-11-12',
			to: '2026-11-14'
		});
		expect(getReportPeriod(new URLSearchParams('from=2026-11-31'))).toEqual(reportPreviewPeriod);
		expect(getReportPeriod(new URLSearchParams('from=2026-11-15&to=2026-11-14'))).toEqual(
			reportPreviewPeriod
		);
	});
});
