import { staffApi, staffApiPath } from './staff.ts';
import type { BookingOverview, BookingReports, ReportPeriod } from '#lib/types/admin-reports.ts';
import type { ApiData } from '#lib/types/identity.ts';

export const staffOverviewEndpoint = `${staffApiPath}/overview`;
export const staffReportsEndpoint = `${staffApiPath}/reports`;

export async function fetchStaffOverview(signal?: AbortSignal) {
	const { data } = await staffApi.get<ApiData<BookingOverview>>(staffOverviewEndpoint, { signal });
	return data.data;
}

export async function fetchStaffReports(period?: ReportPeriod, signal?: AbortSignal) {
	const { data } = await staffApi.get<ApiData<BookingReports>>(staffReportsEndpoint, {
		params: period,
		signal
	});
	return data.data;
}
