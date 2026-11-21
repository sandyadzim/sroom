<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ArrowLeft } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { isAxiosError } from 'axios';
	import { getApiError } from '#lib/api/staff.ts';
	import AdminDetailRow from '#lib/components/admin/AdminDetailRow.svelte';
	import AdminDetailSection from '#lib/components/admin/AdminDetailSection.svelte';
	import StaffCustomerLink from '#lib/components/admin/StaffCustomerLink.svelte';
	import StaffInvoicePanel from '#lib/components/admin/StaffInvoicePanel.svelte';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import TransactionTimeline from '#lib/components/admin/TransactionTimeline.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import {
		bookingPaymentAttemptsQuery,
		staffBookingQuery
	} from '#lib/queries/staff-transactions.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import { bookingTimeline, matchingBookingPayments } from '#lib/utils/staff-transactions.ts';

	const id = $derived(page.params.id ?? '');
	const booking = createQuery(() => staffBookingQuery(id));
	const payments = createQuery(() => ({
		...bookingPaymentAttemptsQuery(id),
		enabled: booking.isSuccess
	}));
	const paymentRows = $derived(payments.data?.data ?? []);
	const attempts = $derived(matchingBookingPayments(paymentRows, id));
	const paymentsTruncated = $derived(
		Boolean(payments.data && payments.data.page.total > paymentRows.length)
	);
	const errorStatus = $derived(
		isAxiosError(booking.error) ? booking.error.response?.status : undefined
	);
	const timeline = $derived(booking.data ? bookingTimeline(booking.data.status_history) : []);
</script>

<svelte:head><title>{booking.data?.id ?? 'Booking details'} — SROOM</title></svelte:head>

<h1 class="sr-only">Booking details</h1>
<Link
	href={resolve('/admin/(staff)/bookings') + page.url.search}
	class="gap-2 text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} aria-hidden="true" /> Back to bookings
</Link>

{#if booking.isPending}
	<p class="mt-6 border-t border-line pt-6 text-ink-muted" role="status">Loading booking…</p>
{:else if booking.isError}
	<div class="mt-6 border-t border-line pt-6">
		<p role="alert" class="text-negative">
			{#if errorStatus === 404 || errorStatus === 400}
				Booking not found.
			{:else if errorStatus === 403}
				Access denied.
			{:else}
				{getApiError(booking.error)?.message ?? 'Could not load this booking.'}
			{/if}
		</p>
		{#if errorStatus !== 404 && errorStatus !== 400 && errorStatus !== 403}
			<Button
				class="mt-3"
				variant="secondary"
				disabled={booking.isFetching}
				onclick={() => booking.refetch()}
			>
				{booking.isFetching ? 'Loading…' : 'Try again'}
			</Button>
		{/if}
	</div>
{:else if booking.data}
	{@const record = booking.data}
	<div class="print:hidden">
		<AdminDetailSection title="Reservation">
			<AdminDetailRow label="Status">
				<TransactionStatus value={{ booking: record.status }} />
			</AdminDetailRow>
			<AdminDetailRow label="Customer">
				<StaffCustomerLink customerId={record.customer_id} />
			</AdminDetailRow>
			<AdminDetailRow label="Vehicle">
				<Link
					href={resolve('/admin/(staff)/cars/[id]', { id: record.variant_id })}
					variant="plain"
					class="text-primary underline"
				>
					{record.vehicle_label}
				</Link>
			</AdminDetailRow>
			<AdminDetailRow label="Physical unit">
				{#if record.unit_id}
					<Link
						href={resolve('/admin/(staff)/units/[id]', { id: record.unit_id })}
						variant="plain"
						class="text-primary underline"
					>
						{record.unit_id}
					</Link>
				{:else}
					Unassigned
				{/if}
			</AdminDetailRow>
			<AdminDetailRow label="Pickup"
				><time datetime={record.pickup_at}>{formatWitaDateTime(record.pickup_at)} WITA</time
				></AdminDetailRow
			>
			<AdminDetailRow label="Return"
				><time datetime={record.return_at}>{formatWitaDateTime(record.return_at)} WITA</time
				></AdminDetailRow
			>
			<AdminDetailRow label="Rate / day">{formatIDR(record.daily_rate_idr)}</AdminDetailRow>
			<AdminDetailRow label="Rental days">{record.rental_days}</AdminDetailRow>
			<AdminDetailRow label="Booking total">{formatIDR(record.total_idr)}</AdminDetailRow>
			<AdminDetailRow label="Hold expires"
				><time datetime={record.hold_expires_at}
					>{formatWitaDateTime(record.hold_expires_at)} WITA</time
				></AdminDetailRow
			>
			<AdminDetailRow label="Created at"
				><time datetime={record.created_at}>{formatWitaDateTime(record.created_at)} WITA</time
				></AdminDetailRow
			>
		</AdminDetailSection>

		<AdminDetailSection title="Payment">
			{#if payments.isPending}
				<AdminDetailRow label="Reference">Loading…</AdminDetailRow>
			{:else if payments.isError}
				<AdminDetailRow label="Reference">
					<p role="alert" class="text-negative">
						{getApiError(payments.error)?.message ?? 'Could not load payment.'}
					</p>
					<Button
						class="mt-3"
						variant="secondary"
						disabled={payments.isFetching}
						onclick={() => payments.refetch()}
					>
						{payments.isFetching ? 'Loading…' : 'Try again'}
					</Button>
				</AdminDetailRow>
			{:else if attempts.length === 0}
				<AdminDetailRow label="Reference">No payment.</AdminDetailRow>
			{:else}
				{#each attempts as payment, index (payment.payment_id)}
					<AdminDetailRow label={index === 0 ? 'Reference' : `Reference ${index + 1}`}>
						<Link
							href={resolve('/admin/(staff)/payments/[id]', { id: payment.payment_id })}
							variant="plain"
							class="text-primary underline"
						>
							{payment.payment_id}
						</Link>
						<span class="mt-2 flex flex-wrap items-center gap-2 text-sm">
							<TransactionStatus value={{ payment: payment.status }} />
							<span class="tabular-nums">{formatIDR(payment.amount_idr)}</span>
							{#if payment.review_required}
								<span class="text-xs font-semibold text-information">Review required</span>
							{/if}
						</span>
					</AdminDetailRow>
				{/each}
			{/if}
			{#if paymentsTruncated}
				<AdminDetailRow label="Note">More payments matched than can be shown here.</AdminDetailRow>
			{/if}
		</AdminDetailSection>

		{#if timeline.length}
			<TransactionTimeline title="Booking timeline" events={timeline} />
		{:else}
			<section class="mt-6 border-t border-line pt-6">
				<h2 class="text-lg/7 font-semibold">Booking timeline</h2>
				<p class="mt-3 text-sm text-ink-muted">No status history recorded.</p>
			</section>
		{/if}
	</div>

	{#if record.status === 'confirmed'}
		<StaffInvoicePanel bookingId={record.id} />
	{/if}
{/if}
