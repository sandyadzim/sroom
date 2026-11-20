import { customerApi, customerPath } from './customer.ts';
import type { ApiData } from '#lib/types/identity.ts';
import type { Booking, HoldInput, QuoteInput, RentalQuote, SavedHold } from '#lib/types/booking.ts';

export async function fetchRentalQuote(input: QuoteInput, signal?: AbortSignal) {
	const { data } = await customerApi.post<ApiData<RentalQuote>>(
		`${customerPath}/checkout/quotes`,
		input,
		{ signal }
	);
	return data.data;
}
export async function createBooking(input: HoldInput, key: string) {
	const { data } = await customerApi.post<ApiData<SavedHold>>(`${customerPath}/bookings`, input, {
		headers: { 'Idempotency-Key': key }
	});
	return data.data;
}
export async function fetchBooking(id: string, signal?: AbortSignal) {
	const { data } = await customerApi.get<ApiData<Booking>>(
		`${customerPath}/bookings/${encodeURIComponent(id)}`,
		{ signal }
	);
	return data.data;
}
