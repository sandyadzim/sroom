export type BookingStatus = 'pending_payment' | 'confirmed' | 'expired' | 'payment_review';
export type PaymentStatus = 'pending' | 'paid' | 'expired' | 'failed';

export type TransactionEvent = {
	title: string;
	time: string;
	description?: string;
};

export type AdminBookingPreview = {
	id: string;
	customer: string;
	model: string;
	unit: string | null;
	pickupAt: string;
	returnAt: string;
	createdAt: string;
	days: number;
	dailyRate: number;
	total: number;
	status: BookingStatus;
	timeline: TransactionEvent[];
};

export type AdminPaymentPreview = {
	id: string;
	bookingId: string;
	provider: 'Midtrans';
	bank: string;
	providerReference: string;
	total: number;
	status: PaymentStatus;
	createdAt: string;
	updatedAt: string;
	reviewReason?: string;
	timeline: TransactionEvent[];
};
