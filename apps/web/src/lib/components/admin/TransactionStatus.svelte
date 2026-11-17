<script lang="ts">
	import { bookingStatusStyles, paymentStatusStyles } from '#lib/utils/admin-transaction-status.ts';
	import type { BookingStatus, PaymentStatus } from '#lib/types/admin-transactions.ts';

	let { value }: { value: { booking: BookingStatus } | { payment: PaymentStatus } } = $props();
	const status = $derived(
		'booking' in value ? bookingStatusStyles[value.booking] : paymentStatusStyles[value.payment]
	);
</script>

<span
	class={[
		'inline-flex rounded-full px-2.5 py-1 text-xs font-semibold whitespace-nowrap',
		status.tone
	]}
>
	{status.label}
</span>
