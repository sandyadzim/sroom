export type BookingStatus = 'pending_payment' | 'confirmed' | 'expired' | 'payment_review';
export type QuoteInput = { variant_id: string; pickup_at: string; return_at: string };
export type RentalQuote = QuoteInput & {
	vehicle_label: string;
	rental_days: number;
	daily_rate_idr: number;
	total_idr: number;
	currency: 'IDR';
	available: boolean;
	quoted_at: string;
};
export type HoldInput = QuoteInput & {
	expected_total_idr: number;
	terms_version: 'rental-terms-v1';
	accept_terms: true;
};
export type Booking = QuoteInput & {
	id: string;
	vehicle_label: string;
	status: BookingStatus;
	rental_days: number;
	daily_rate_idr: number;
	total_idr: number;
	currency: 'IDR';
	hold_expires_at: string;
	terms_version: string;
	terms_accepted_at: string;
	created_at: string;
};
export type SavedHold = Omit<Booking, 'terms_version' | 'terms_accepted_at' | 'created_at'> & {
	payment_status: 'not_started';
};
export type PendingHold = { key: string; payload: HoldInput };

export type BookingHistoryFilters = { status: BookingStatus | 'all'; page: number };
export type CustomerBookingsInput = { status?: BookingStatus; page: number; limit: number };
export type CustomerBookingsPage = {
	data: Booking[];
	page: { number: number; limit: number; total: number };
};
