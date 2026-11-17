import { z } from 'zod';

export const checkoutPreviewSlug = 'toyota-fortuner';
export const checkoutPreviewImage = 'images/hero/fortuner-studio.webp';
export const checkoutPreviewCustomer = { name: 'Alex Morgan', email: 'alex@example.invalid' };

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
	createdAt: z.number().int().positive().optional(),
	outcomeAt: z.number().int().positive().optional(),
	nextOutcome: outcomeSchema.optional(),
	outcome: outcomeSchema.optional()
});

export type BookingPreview = z.infer<typeof bookingSchema>;

const storagePrefix = 'sroom-booking-preview:';
const holdDurationMs = 15 * 60_000;

export function saveBookingPreview(
	input: Omit<BookingPreview, 'expiresAt' | 'createdAt' | 'outcomeAt' | 'outcome' | 'nextOutcome'>,
	nextOutcome?: BookingOutcome
) {
	const id = `preview-${crypto.randomUUID()}`;
	const createdAt = Date.now();
	sessionStorage.setItem(
		storagePrefix + id,
		JSON.stringify({
			...input,
			createdAt,
			expiresAt: createdAt + (nextOutcome === 'expired' ? 0 : holdDurationMs),
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

export function listBookingPreviews() {
	const keys = Array.from({ length: sessionStorage.length }, (_, index) =>
		sessionStorage.key(index)
	);
	const createdAt = (booking: BookingPreview) =>
		booking.createdAt ??
		booking.expiresAt - (booking.nextOutcome === 'expired' ? 0 : holdDurationMs);
	return keys
		.flatMap((key) => {
			if (!key?.startsWith(storagePrefix)) return [];
			const id = key.slice(storagePrefix.length);
			if (!isBookingPreviewId(id)) return [];
			const booking = readBookingPreview(id);
			return booking ? [{ id, ...booking }] : [];
		})
		.sort((a, b) => createdAt(b) - createdAt(a));
}

export function bookingPreviewPaymentReference(id: string) {
	return `VA-${id.slice(-12).toUpperCase()}`;
}

export function checkBookingPreview(id: string) {
	const booking = readBookingPreview(id);
	if (!booking) throw new Error('Preview booking unavailable.');
	if (booking.outcome || !booking.nextOutcome) return booking.outcome;

	const outcome = booking.nextOutcome;
	sessionStorage.setItem(
		storagePrefix + id,
		JSON.stringify({ ...booking, outcome, outcomeAt: Date.now() })
	);
	return outcome;
}

export function isBookingPreviewId(id: string) {
	return /^preview-[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(id);
}

export function remainingPaymentSeconds(expiresAt: number, now: number) {
	return Math.max(0, Math.ceil((expiresAt - now) / 1000));
}

export function addBookingHistorySamples() {
	const now = Date.now();
	const samples = [
		{ id: 'preview-00000000-0000-4000-8000-000000000001', outcome: undefined, days: 3 },
		{ id: 'preview-00000000-0000-4000-8000-000000000002', outcome: 'confirmed', days: 2 },
		{ id: 'preview-00000000-0000-4000-8000-000000000003', outcome: 'expired', days: 4 },
		{ id: 'preview-00000000-0000-4000-8000-000000000004', outcome: 'review', days: 1 }
	] as const;

	for (const [index, sample] of samples.entries()) {
		const key = storagePrefix + sample.id;
		if (sessionStorage.getItem(key)) continue;

		const createdAt = now - index * 86_400_000;
		const pickupDate = new Date(now + (3 + index) * 86_400_000).toISOString().slice(0, 10);
		const returnDate = new Date(now + (2 + index + sample.days) * 86_400_000)
			.toISOString()
			.slice(0, 10);
		const dailyRate = 850_000;
		const expiresAt = createdAt + holdDurationMs;
		const outcomeAt = sample.outcome
			? sample.outcome === 'expired'
				? expiresAt
				: createdAt + 300_000
			: undefined;
		const booking: BookingPreview = {
			carName: 'Toyota Fortuner',
			carTrim: '2.8 VRZ',
			pickupAt: `${pickupDate}T09:00:00+08:00`,
			returnAt: `${returnDate}T17:00:00+08:00`,
			days: sample.days,
			dailyRate,
			total: dailyRate * sample.days,
			createdAt,
			expiresAt,
			outcome: sample.outcome,
			outcomeAt
		};
		sessionStorage.setItem(key, JSON.stringify(booking));
	}
}
