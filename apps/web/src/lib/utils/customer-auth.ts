import { bookingHistorySearch, readBookingFilters } from './booking-history.ts';
type CustomerReturnPath =
	| '/account/bookings'
	| `/account/bookings?${string}`
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
const bookingHistoryPath = new RegExp(`^/account/bookings(?:/${uuidPattern}(?:/invoice)?)?$`);
export function customerReturnPath(next: string | null) {
	if (!next || /[\s\\#]/.test(next)) return undefined;
	const [pathname, search, extra] = next.split('?');
	if (pathname.includes('%') || extra !== undefined) return undefined;
	if (bookingHistoryPath.test(pathname)) {
		try {
			decodeURIComponent(search ?? '');
		} catch {
			return undefined;
		}
		const params = new URLSearchParams(search);
		if ([...params.keys()].some((key) => key !== 'status' && key !== 'page')) return undefined;
		const selection = readBookingFilters(params);
		return `${pathname}${bookingHistorySearch(selection.status, selection.page)}` as CustomerReturnPath;
	}
	if (search !== undefined) return undefined;
	if (customerAccountPath(pathname) || checkoutPath.test(pathname) || rentalPath.test(pathname))
		return pathname as CustomerReturnPath;
}
export function isBookingId(id: string) {
	return new RegExp(`^${uuidPattern}$`).test(id);
}
