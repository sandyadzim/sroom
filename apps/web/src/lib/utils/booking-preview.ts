import { z } from 'zod';

export const checkoutPreviewSlug = 'toyota-fortuner';
export const checkoutPreviewImage = 'images/hero/fortuner-studio.webp';

const outcomeSchema = z.enum(['confirmed', 'expired', 'review']);
export type BookingOutcome = z.infer<typeof outcomeSchema>;

const outcomeRoutes = { confirmed: 'success', expired: 'expired', review: 'review' } as const;

export function bookingOutcomeRoute(outcome: BookingOutcome) {
	return outcomeRoutes[outcome];
}

export function bookingOutcomeFixture(value: string | null) {
	return outcomeSchema.safeParse(value).data;
}

const bookingSchema = z.object({
	carName: z.string(),
	carTrim: z.string(),
	pickupAt: z.iso.datetime({ offset: true }),
	returnAt: z.iso.datetime({ offset: true }),
	days: z.number().int().positive(),
	dailyRate: z.number().int().nonnegative(),
	total: z.number().int().nonnegative(),
	expiresAt: z.number().int().positive(),
	nextOutcome: outcomeSchema.optional(),
	outcome: outcomeSchema.optional()
});

export type BookingPreview = z.infer<typeof bookingSchema>;

const storagePrefix = 'sroom-booking-preview:';
const holdDurationMs = 15 * 60_000;

export function saveBookingPreview(
	input: Omit<BookingPreview, 'expiresAt' | 'outcome' | 'nextOutcome'>,
	nextOutcome?: BookingOutcome
) {
	const id = `preview-${crypto.randomUUID()}`;
	sessionStorage.setItem(
		storagePrefix + id,
		JSON.stringify({
			...input,
			expiresAt: Date.now() + (nextOutcome === 'expired' ? 0 : holdDurationMs),
			nextOutcome
		})
	);
	return id;
}

export function readBookingPreview(id: string) {
	try {
		const value = sessionStorage.getItem(storagePrefix + id);
		if (!value) return null;
		return bookingSchema.safeParse(JSON.parse(value)).data ?? null;
	} catch {
		return null;
	}
}

export function checkBookingPreview(id: string) {
	const booking = readBookingPreview(id);
	if (!booking) throw new Error('Preview booking unavailable.');
	if (booking.outcome || !booking.nextOutcome) return booking.outcome;

	const outcome = booking.nextOutcome;
	sessionStorage.setItem(storagePrefix + id, JSON.stringify({ ...booking, outcome }));
	return outcome;
}

export function isBookingPreviewId(id: string) {
	return /^preview-[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(id);
}

export function remainingPaymentSeconds(expiresAt: number, now: number) {
	return Math.max(0, Math.ceil((expiresAt - now) / 1000));
}
