<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ArrowLeft } from '@lucide/svelte';
	import AdminDetailRow from '#lib/components/admin/AdminDetailRow.svelte';
	import AdminDetailSection from '#lib/components/admin/AdminDetailSection.svelte';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import TransactionTimeline from '#lib/components/admin/TransactionTimeline.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { findBookingPreview } from '#lib/utils/admin-transaction-preview.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	const record = $derived(findBookingPreview(page.params.id ?? ''));
</script>

<svelte:head><title>{record?.booking.id ?? 'Booking details'} — SROOM</title></svelte:head>

<Link
	href={resolve('/admin/(staff)/bookings') + page.url.search}
	class="gap-2 text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} aria-hidden="true" /> Back to bookings
</Link>

{#if record}
	{@const { booking, payment } = record}
	<header class="mt-2">
		<h1 class="text-3xl/10 font-bold tracking-tight wrap-anywhere sm:text-4xl/12">{booking.id}</h1>
		<div class="mt-3 flex flex-wrap items-center gap-3">
			<TransactionStatus value={{ booking: booking.status }} />
			<p class="text-sm text-ink-muted">Booking details · Preview data</p>
		</div>
	</header>

	<AdminDetailSection title="Reservation">
		<AdminDetailRow label="Customer">{booking.customer}</AdminDetailRow>
		<AdminDetailRow label="Vehicle">{booking.model}</AdminDetailRow>
		<AdminDetailRow label="Physical unit">{booking.unit ?? 'Unassigned'}</AdminDetailRow>
		<AdminDetailRow label="Pickup"
			><time datetime={booking.pickupAt}>{formatWitaDateTime(booking.pickupAt)} WITA</time
			></AdminDetailRow
		>
		<AdminDetailRow label="Return"
			><time datetime={booking.returnAt}>{formatWitaDateTime(booking.returnAt)} WITA</time
			></AdminDetailRow
		>
		<AdminDetailRow label="Rate / day">{formatIDR(booking.dailyRate)}</AdminDetailRow>
		<AdminDetailRow label="Rental days">{booking.days}</AdminDetailRow>
		<AdminDetailRow label="Booking total">{formatIDR(booking.total)}</AdminDetailRow>
	</AdminDetailSection>

	<AdminDetailSection title="Payment">
		{#if payment}
			<AdminDetailRow label="Payment status"
				><TransactionStatus value={{ payment: payment.status }} /></AdminDetailRow
			>
			<AdminDetailRow label="Payment attempt">
				<Link
					href={resolve('/admin/(staff)/payments/[id]', { id: payment.id })}
					variant="plain"
					class="text-primary underline">{payment.id}</Link
				>
			</AdminDetailRow>
			{#if payment.reviewReason}<AdminDetailRow label="Reconciliation"
					>{payment.reviewReason}</AdminDetailRow
				>{/if}
		{:else}
			<AdminDetailRow label="Payment status">No payment attempt</AdminDetailRow>
		{/if}
	</AdminDetailSection>

	<TransactionTimeline title="Booking timeline" events={booking.timeline} />
{:else}
	<h1 class="mt-2 text-3xl font-bold tracking-tight">Booking not found</h1>
	<p class="mt-2 text-sm text-ink-muted">Return to Bookings to choose an available record.</p>
{/if}
