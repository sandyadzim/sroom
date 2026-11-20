export type PaymentStatus =
	'not_started' | 'initiating' | 'pending' | 'paid' | 'expired' | 'failed';
export type CustomerPayment = {
	payment_id: string | null;
	booking_id: string;
	booking_status: string;
	status: PaymentStatus;
	amount_idr: number;
	currency: 'IDR';
	bank: 'bni' | null;
	va_number: string | null;
	hold_expires_at: string;
	provider_expires_at: string | null;
	verified_at: string | null;
	provider_paid_at: string | null;
	review_required: boolean;
};
export type VirtualAccount = Omit<
	CustomerPayment,
	'booking_status' | 'verified_at' | 'provider_paid_at'
> & { payment_id: string; bank: 'bni' };
export type Invoice = {
	id: string;
	booking_id: string;
	payment_id: string;
	customer_id: string;
	variant_id: string;
	invoice_number: string;
	provider_order_id: string;
	vehicle_label: string;
	pickup_at: string;
	return_at: string;
	confirmed_at: string;
	issued_at: string;
	rental_days: number;
	daily_rate_idr: number;
	total_idr: number;
	currency: 'IDR';
	bank: 'bni';
	payment_method: 'virtual_account';
	status: 'paid';
	document_type: 'non_tax_invoice';
	billed_to: { full_name: string; email: string };
};
