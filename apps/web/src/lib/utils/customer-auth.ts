const checkoutPath = /^\/checkout\/([a-z0-9]+(?:-[a-z0-9]+)*)$/;

export function customerCheckoutSlug(next: string | null) {
	return next?.match(checkoutPath)?.[1];
}
