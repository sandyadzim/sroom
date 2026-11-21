export type BookingStatus = 'pending_payment' | 'confirmed' | 'expired' | 'payment_review';
export type PaymentStatus = 'initiating' | 'pending' | 'paid' | 'expired' | 'failed';

export type TransactionEvent = {
	title: string;
	time: string;
	description?: string;
};

export type StaffBooking = {
	id: string;
	variant_id: string;
	vehicle_label: string;
	status: BookingStatus;
	pickup_at: string;
	return_at: string;
	rental_days: number;
	daily_rate_idr: number;
	total_idr: number;
	currency: 'IDR';
	hold_expires_at: string;
	terms_version: string;
	terms_accepted_at: string;
	created_at: string;
	customer_id: string;
	unit_id: string | null;
};

export type BookingStatusHistoryEntry = {
	id: string;
	from_status: string | null;
	to_status: BookingStatus;
	actor_type: 'system' | 'customer' | 'admin' | 'provider_event';
	reason_code: string | null;
	occurred_at: string;
};

export type StaffBookingDetail = StaffBooking & {
	status_history: BookingStatusHistoryEntry[];
};

export type StaffBookingRow = StaffBooking & { customer_name: string | null };

export type StaffPayment = {
	payment_id: string;
	booking_id: string;
	customer_id: string;
	provider: 'midtrans';
	provider_order_id: string;
	provider_transaction_id: string | null;
	status: PaymentStatus;
	bank: 'bni';
	amount_idr: number;
	currency: 'IDR';
	review_required: boolean;
	review_reason: string | null;
	hold_expires_at: string;
	expiry_review_required: boolean;
	created_at: string;
	updated_at: string;
};

export type PaymentTimelineEntry = {
	id: string;
	source: 'created' | 'charge' | 'webhook' | 'status_lookup';
	occurred_at: string;
	provider_status: string | null;
	verification_result: 'verified' | 'rejected' | 'unavailable' | null;
	error_code: string | null;
};

export type StaffPaymentDetail = StaffPayment & {
	booking_status: string;
	provider_status: string | null;
	provider_expires_at: string | null;
	verified_at: string | null;
	provider_paid_at: string | null;
	timeline: PaymentTimelineEntry[];
};

export type StaffPaymentRow = StaffPayment & { customer_name: string | null };
