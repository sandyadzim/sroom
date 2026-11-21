import { queryOptions } from '@tanstack/svelte-query';
import { fetchStaffOverview, fetchStaffReports } from '#lib/api/staff-reports.ts';
import type { ReportPeriod } from '#lib/types/admin-reports.ts';

export const staffReportsKey = ['staff-reports'] as const;

export function staffOverviewQuery() {
	return queryOptions({
		queryKey: [...staffReportsKey, 'overview'],
		queryFn: ({ signal }) => fetchStaffOverview(signal),
		retry: false,
		staleTime: 0
	});
}

export function staffReportsQuery(period?: ReportPeriod) {
	return queryOptions({
		queryKey: [...staffReportsKey, 'reports', period ?? 'default'],
		queryFn: ({ signal }) => fetchStaffReports(period, signal),
		retry: false,
		staleTime: 0
	});
}
