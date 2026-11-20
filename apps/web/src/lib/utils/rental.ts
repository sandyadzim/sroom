import type { Booking, HoldInput, PendingHold } from '#lib/types/booking.ts';
import type { CustomerPayment } from '#lib/types/payment.ts';

export function rentalOutcome(booking: Booking, payment?: CustomerPayment) {
	if (booking.status === 'confirmed') return 'success';
	if (booking.status === 'expired') return 'expired';
	if (booking.status === 'payment_review' || payment?.review_required) return 'review';
	return undefined;
}
export function paymentDeadline(
	payment: Pick<CustomerPayment, 'hold_expires_at' | 'provider_expires_at'>
) {
	const hold = Date.parse(payment.hold_expires_at);
	return payment.provider_expires_at
		? Math.min(hold, Date.parse(payment.provider_expires_at))
		: hold;
}
export function canShowVirtualAccount(payment: CustomerPayment, now: number) {
	return (
		payment.status === 'pending' &&
		!payment.review_required &&
		!!payment.va_number &&
		now < paymentDeadline(payment)
	);
}
const storagePrefix = 'sroom-rental:';
export function readPendingHold(customerId: string, slug: string): PendingHold | undefined {
	try {
		const saved = JSON.parse(
			sessionStorage.getItem(`${storagePrefix}${customerId}:hold:${slug}`) ?? 'null'
		);
		if (
			typeof saved?.key === 'string' &&
			typeof saved?.payload?.variant_id === 'string' &&
			typeof saved.payload.pickup_at === 'string' &&
			typeof saved.payload.return_at === 'string' &&
			Number.isSafeInteger(saved.payload.expected_total_idr) &&
			saved.payload.expected_total_idr > 0 &&
			saved.payload.terms_version === 'rental-terms-v1' &&
			saved.payload.accept_terms === true
		)
			return saved;
	} catch {
		/* Missing browser storage leaves the form empty. */
	}
}
export function holdCommand(customerId: string, slug: string, payload: HoldInput): PendingHold {
	const saved = readPendingHold(customerId, slug);
	if (saved && JSON.stringify(saved.payload) === JSON.stringify(payload)) return saved;
	const command = { key: crypto.randomUUID(), payload };
	sessionStorage.setItem(`${storagePrefix}${customerId}:hold:${slug}`, JSON.stringify(command));
	return command;
}
export function virtualAccountKey(customerId: string, bookingId: string) {
	const storageKey = `${storagePrefix}${customerId}:va:${bookingId}`;
	const key = sessionStorage.getItem(storageKey) ?? crypto.randomUUID();
	sessionStorage.setItem(storageKey, key);
	return key;
}

export function clearPendingHold(customerId: string, slug: string) {
	sessionStorage.removeItem(`${storagePrefix}${customerId}:hold:${slug}`);
}
