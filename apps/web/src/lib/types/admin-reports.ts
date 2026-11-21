import type { StaffBooking } from '#lib/types/admin-transactions.ts';

export type ReportPeriod = {
	from: string;
	to: string;
};

export type ReportSummary = {
	total_bookings: number;
	pending_payment_bookings: number;
	confirmed_bookings: number;
	expired_bookings: number;
	payment_review_bookings: number;
	verified_confirmed_bookings: number;
	confirmed_value_idr: number;
	payment_conversion: number | null;
};

export type ReportDay = {
	date: string;
	bookings: number;
	confirmed_value_idr: number;
};

export type ReportVariant = {
	variant_id: string;
	vehicle_label: string;
	bookings: number;
};

export type BookingReportBase = {
	period: ReportPeriod;
	timezone: 'Asia/Makassar';
	currency: 'IDR';
	simulated: true;
	summary: ReportSummary;
	daily: ReportDay[];
};

export type BookingReports = BookingReportBase & {
	top_variants: ReportVariant[];
};

export type FleetCounts = {
	total: number;
	active: number;
	inactive: number;
};

export type FleetSummary = {
	cars: FleetCounts;
	units: FleetCounts;
};

export type BookingOverview = BookingReportBase & {
	recent_bookings: StaffBooking[];
	fleet: FleetSummary;
};
