type CustomerReturnPath =
	| '/account/profile'
	| '/account/profile/change-password'
	| `/checkout/${string}`
	| `/payment/${string}`
	| `/bookings/${string}/${'success' | 'expired' | 'review'}`
	| `/account/bookings/${string}`;
const slugPattern = '[a-z0-9]+(?:-[a-z0-9]+)*';
const uuidPattern = '[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}';
const checkoutPath = new RegExp(`^/checkout/(${slugPattern})$`);
const rentalPath = new RegExp(
	`^/(?:payment/${uuidPattern}|bookings/${uuidPattern}/(?:success|expired|review)|account/bookings/${uuidPattern}(?:/invoice)?)$`
);
export function customerCheckoutSlug(next: string | null) {
	return next && !/[\s%?#\\]/.test(next) ? next.match(checkoutPath)?.[1] : undefined;
}
export function customerAccountPath(next: string | null) {
	if (next === '/account/profile' || next === '/account/profile/change-password') return next;
}
export function customerReturnPath(next: string | null) {
	if (!next || /[\s%?#\\]/.test(next)) return undefined;
	if (customerAccountPath(next) || checkoutPath.test(next) || rentalPath.test(next))
		return next as CustomerReturnPath;
}
export function isBookingId(id: string) {
	return new RegExp(`^${uuidPattern}$`).test(id);
}
