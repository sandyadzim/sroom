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
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import TransactionTimeline from '#lib/components/admin/TransactionTimeline.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { staffPaymentQuery } from '#lib/queries/staff-transactions.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import { isBookingStatus, paymentTimeline } from '#lib/utils/staff-transactions.ts';

	const id = $derived(page.params.id ?? '');
	const payment = createQuery(() => staffPaymentQuery(id));
	const errorStatus = $derived(
		isAxiosError(payment.error) ? payment.error.response?.status : undefined
	);
	const timeline = $derived(payment.data ? paymentTimeline(payment.data.timeline) : []);
</script>

<svelte:head><title>{payment.data?.payment_id ?? 'Payment details'} — SROOM</title></svelte:head>

<h1 class="sr-only">Payment details</h1>
<Link
	href={resolve('/admin/(staff)/payments') + page.url.search}
	class="gap-2 text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} aria-hidden="true" /> Back to payments
</Link>

{#if payment.isPending}
	<p class="mt-6 border-t border-line pt-6 text-ink-muted" role="status">Loading payment…</p>
{:else if payment.isError}
	<div class="mt-6 border-t border-line pt-6">
		<p role="alert" class="text-negative">
			{#if errorStatus === 404 || errorStatus === 400}
				Payment not found.
			{:else if errorStatus === 403}
				Access denied.
			{:else}
				{getApiError(payment.error)?.message ?? 'Could not load this payment.'}
			{/if}
		</p>
		{#if errorStatus !== 404 && errorStatus !== 400 && errorStatus !== 403}
			<Button
				class="mt-3"
				variant="secondary"
				disabled={payment.isFetching}
				onclick={() => payment.refetch()}
			>
				{payment.isFetching ? 'Loading…' : 'Try again'}
			</Button>
		{/if}
	</div>
{:else if payment.data}
	{@const record = payment.data}
	<AdminDetailSection title="Payment">
		<AdminDetailRow label="Status">
			<TransactionStatus value={{ payment: record.status }} />
		</AdminDetailRow>
		{#if record.review_required}
			<AdminDetailRow label="Reconciliation">
				<span class="text-information">
					Review required{#if record.review_reason}: {record.review_reason}{/if}
				</span>
			</AdminDetailRow>
		{/if}
		{#if record.expiry_review_required}
			<AdminDetailRow label="Hold deadline">
				<span class="text-caution">
					The hold deadline has passed. Verify the provider outcome before taking further action.
				</span>
			</AdminDetailRow>
		{/if}
		<AdminDetailRow label="Provider"
			>{record.provider === 'midtrans' ? 'Midtrans' : record.provider}</AdminDetailRow
		>
		<AdminDetailRow label="Method">Virtual Account · {record.bank.toUpperCase()}</AdminDetailRow>
		<AdminDetailRow label="Provider order ID">{record.provider_order_id}</AdminDetailRow>
		<AdminDetailRow label="Provider transaction ID"
			>{record.provider_transaction_id ?? '—'}</AdminDetailRow
		>
		<AdminDetailRow label="Amount">{formatIDR(record.amount_idr)}</AdminDetailRow>
		<AdminDetailRow label="Created at"
			><time datetime={record.created_at}>{formatWitaDateTime(record.created_at)} WITA</time
			></AdminDetailRow
		>
		<AdminDetailRow label="Last updated"
			><time datetime={record.updated_at}>{formatWitaDateTime(record.updated_at)} WITA</time
			></AdminDetailRow
		>
		<AdminDetailRow label="Hold expires"
			><time datetime={record.hold_expires_at}
				>{formatWitaDateTime(record.hold_expires_at)} WITA</time
			></AdminDetailRow
		>
		<AdminDetailRow label="Provider expires">
			{#if record.provider_expires_at}
				<time datetime={record.provider_expires_at}
					>{formatWitaDateTime(record.provider_expires_at)} WITA</time
				>
			{:else}
				—
			{/if}
		</AdminDetailRow>
		<AdminDetailRow label="Verified at">
			{#if record.verified_at}
				<time datetime={record.verified_at}>{formatWitaDateTime(record.verified_at)} WITA</time>
			{:else}
				—
			{/if}
		</AdminDetailRow>
		<AdminDetailRow label="Provider paid at">
			{#if record.provider_paid_at}
				<time datetime={record.provider_paid_at}
					>{formatWitaDateTime(record.provider_paid_at)} WITA</time
				>
			{:else}
				—
			{/if}
		</AdminDetailRow>
		<AdminDetailRow label="Provider status">{record.provider_status ?? '—'}</AdminDetailRow>
	</AdminDetailSection>

	<AdminDetailSection title="Related booking">
		<AdminDetailRow label="Booking">
			<Link
				href={resolve('/admin/(staff)/bookings/[id]', { id: record.booking_id })}
				variant="plain"
				class="text-primary underline">{record.booking_id}</Link
			>
		</AdminDetailRow>
		<AdminDetailRow label="Booking status">
			{#if isBookingStatus(record.booking_status)}
				<TransactionStatus value={{ booking: record.booking_status }} />
			{:else}
				{record.booking_status}
			{/if}
		</AdminDetailRow>
		<AdminDetailRow label="Customer">
			<StaffCustomerLink customerId={record.customer_id} />
		</AdminDetailRow>
	</AdminDetailSection>

	{#if timeline.length}
		<TransactionTimeline title="Payment timeline" events={timeline} />
	{:else}
		<section class="mt-6 border-t border-line pt-6">
			<h2 class="text-lg/7 font-semibold">Payment timeline</h2>
			<p class="mt-3 text-sm text-ink-muted">No provider observations recorded.</p>
		</section>
	{/if}
{/if}
