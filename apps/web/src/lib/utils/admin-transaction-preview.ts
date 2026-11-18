import type {
	AdminBookingPreview,
	AdminPaymentPreview,
	BookingStatus,
	PaymentStatus
} from '#lib/types/admin-transactions.ts';
import type { TablePage, TableParams } from '#lib/types/data-table.ts';

const fleet = [
	{ name: 'Toyota Fortuner', rate: 850_000, code: 'FORT' },
	{ name: 'Honda Brio', rate: 350_000, code: 'BRIO' },
	{ name: 'Toyota Innova Zenix', rate: 650_000, code: 'ZENIX' },
	{ name: 'Suzuki Jimny', rate: 750_000, code: 'JIMNY' }
] as const;
const unitNumbers = ['01', '02', '03'] as const;
export const previewFleetCounts = {
	models: fleet.length,
	units: fleet.length * unitNumbers.length
};

const cases: ReadonlyArray<{ booking: BookingStatus; payment: PaymentStatus; reason?: string }> = [
	{ booking: 'confirmed', payment: 'paid' },
	{
		booking: 'payment_review',
		payment: 'paid',
		reason: 'Settlement arrived after the reservation was released.'
	},
	{ booking: 'pending_payment', payment: 'pending' },
	{ booking: 'expired', payment: 'expired' },
	{ booking: 'expired', payment: 'failed' },
	{
		booking: 'payment_review',
		payment: 'pending',
		reason: 'Provider status could not be verified.'
	}
];

function witaDay(date: number, time: string) {
	return `2026-11-${String(date).padStart(2, '0')}T${time}:00+08:00`;
}

const bookingDays = [3, 4, 1, 5, 2, 4, 3, 2].flatMap((count, index) =>
	Array<number>(count).fill(17 - index)
);

export const adminTransactionPreviews = bookingDays.map((date, index) => {
	const vehicle = fleet[index % fleet.length];
	const fixture = cases[index % cases.length];
	const number = String(bookingDays.length - index).padStart(3, '0');
	const createdAt = witaDay(date, '09:00');
	const hasPayment = index !== 22;
	const paidAfterExpiry = fixture.payment === 'paid' && fixture.booking === 'payment_review';
	const updatedAt = witaDay(
		date,
		fixture.payment === 'paid'
			? paidAfterExpiry
				? '09:24'
				: '09:08'
			: fixture.payment === 'expired'
				? '09:15'
				: '09:04'
	);
	const days = (index % 3) + 1;
	const activeAllocation =
		fixture.booking === 'confirmed' ||
		fixture.booking === 'pending_payment' ||
		(fixture.booking === 'payment_review' && !paidAfterExpiry);
	const booking: AdminBookingPreview = {
		id: `BKG-202611-${number}`,
		customer: ['Alex Morgan', 'Rani Putri', 'Dimas Pratama', 'Maya Santoso'][index % 4],
		model: vehicle.name,
		unit: activeAllocation ? `${vehicle.code}-${unitNumbers[index % unitNumbers.length]}` : null,
		pickupAt: witaDay(date + 3, '10:00'),
		returnAt: witaDay(date + 2 + days, '17:00'),
		days,
		dailyRate: vehicle.rate,
		total: days * vehicle.rate,
		createdAt,
		status: fixture.booking,
		timeline: [{ title: 'Reservation created', time: createdAt }]
	};

	if (paidAfterExpiry)
		booking.timeline.push({ title: 'Reservation released', time: witaDay(date, '09:15') });
	if (fixture.booking !== 'pending_payment') {
		booking.timeline.push({
			title: {
				confirmed: 'Booking confirmed',
				expired: 'Reservation expired',
				payment_review: 'Booking flagged for review'
			}[fixture.booking],
			time: fixture.booking === 'expired' ? witaDay(date, '09:15') : updatedAt,
			description: fixture.reason
		});
	}

	const payment: AdminPaymentPreview | null = hasPayment
		? {
				id: `PAY-202611-${number}`,
				bookingId: booking.id,
				provider: 'Midtrans',
				bank: ['BCA', 'BNI', 'BRI', 'Mandiri'][index % 4],
				providerReference: `•••• ${3300 + index}`,
				total: booking.total,
				status: fixture.payment,
				createdAt,
				updatedAt,
				reviewReason: fixture.reason,
				timeline: [
					{ title: 'Payment attempt created', time: createdAt },
					{
						title: {
							pending: 'Awaiting settlement',
							paid: 'Settlement verified',
							expired: 'Provider expiry recorded',
							failed: 'Attempt failed'
						}[fixture.payment],
						time: updatedAt
					},
					...(fixture.reason
						? [{ title: 'Reconciliation required', time: updatedAt, description: fixture.reason }]
						: [])
				]
			}
		: null;
	return { booking, payment };
});

export const bookingModels = fleet.map(({ name }) => ({ label: name, value: name }));

export function findBookingPreview(id: string) {
	return adminTransactionPreviews.find(({ booking }) => booking.id === id);
}

export function findPaymentPreview(id: string) {
	return adminTransactionPreviews.find(({ payment }) => payment?.id === id);
}

function tablePage<Row>(rows: Row[], { page, limit }: TableParams): TablePage<Row> {
	const offset = (page - 1) * limit;
	return {
		data: rows.slice(offset, offset + limit),
		page: { number: page, limit, total: rows.length }
	};
}

export function listAdminBookingPreviews(params: TableParams) {
	const { search, status, model, pickup_date } = params.filters;
	const term = search?.toLowerCase().trim();
	const rows = adminTransactionPreviews
		.map(({ booking }) => booking)
		.filter(
			(booking) =>
				(!term ||
					[booking.id, booking.customer, booking.model, booking.unit ?? ''].some((value) =>
						value.toLowerCase().includes(term)
					)) &&
				(!status || booking.status === status) &&
				(!model || booking.model === model) &&
				(!pickup_date || booking.pickupAt.startsWith(`${pickup_date}T`))
		);
	return tablePage(rows, params);
}

export function listAdminPaymentPreviews(params: TableParams) {
	const { search, status, reconciliation } = params.filters;
	const term = search?.toLowerCase().trim();
	const rows = adminTransactionPreviews
		.flatMap(({ payment }) => (payment ? [payment] : []))
		.filter(
			(payment) =>
				(!term ||
					[payment.id, payment.bookingId, payment.providerReference].some((value) =>
						value.toLowerCase().includes(term)
					)) &&
				(!status || payment.status === status) &&
				(!reconciliation || Boolean(payment.reviewReason) === (reconciliation === 'required'))
		);
	return tablePage(rows, params);
}
