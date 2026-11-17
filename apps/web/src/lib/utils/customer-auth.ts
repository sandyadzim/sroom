const checkoutPath = /^\/checkout\/([a-z0-9]+(?:-[a-z0-9]+)*)$/;

export function customerCheckoutSlug(next: string | null) {
	return next?.match(checkoutPath)?.[1];
}

export function customerAccountPath(next: string | null) {
	if (next === '/account/profile' || next === '/account/profile/change-password') return next;
}
