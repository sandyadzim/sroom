<script lang="ts">
	import { bookingStatuses, type BookingStatus } from '#lib/utils/booking-history.ts';

	let { status }: { status: BookingStatus } = $props();
	const style = $derived(bookingStatuses[status]);
</script>

<span
	class={['inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold', style.tone]}
>
	{style.label}
</span>
