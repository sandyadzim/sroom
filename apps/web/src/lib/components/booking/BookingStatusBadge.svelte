<script lang="ts">
	import type { BookingStatus } from '#lib/types/booking.ts';
	import { bookingStatuses } from '#lib/utils/booking-history.ts';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	let { status }: { status: BookingStatus } = $props();
</script>

<StatusBadge {...bookingStatuses[status]} />
