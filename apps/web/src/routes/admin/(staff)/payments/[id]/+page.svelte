<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ArrowLeft } from '@lucide/svelte';
	import AdminDetailRow from '#lib/components/admin/AdminDetailRow.svelte';
	import AdminDetailSection from '#lib/components/admin/AdminDetailSection.svelte';
	import TransactionStatus from '#lib/components/admin/TransactionStatus.svelte';
	import TransactionTimeline from '#lib/components/admin/TransactionTimeline.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { findPaymentPreview } from '#lib/utils/admin-transaction-preview.ts';
	import { formatIDR } from '#lib/utils/currency.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	const record = $derived(findPaymentPreview(page.params.id ?? ''));
	const payment = $derived(record?.payment);
</script>

<svelte:head><title>{payment?.id ?? 'Payment details'} — SROOM</title></svelte:head>

<Link
	href={resolve('/admin/(staff)/payments') + page.url.search}
	class="gap-2 text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} aria-hidden="true" /> Back to payments
</Link>

{#if record && payment}
	<header class="mt-2">
		<h1 class="text-3xl/10 font-bold tracking-tight wrap-anywhere sm:text-4xl/12">{payment.id}</h1>
		<div class="mt-3 flex flex-wrap items-center gap-3">
			<TransactionStatus value={{ payment: payment.status }} />
			<p class="text-sm text-ink-muted">Payment attempt · Preview data</p>
		</div>
	</header>

	{#if payment.reviewReason}
		<section
			aria-labelledby="reconciliation-heading"
			class="mt-5 rounded-xl border border-information/20 bg-information/5 px-4 py-3 sm:px-5"
		>
			<h2 id="reconciliation-heading" class="text-sm font-semibold text-information">
				Review required
			</h2>
			<p class="mt-1 text-sm leading-6 text-ink-muted">{payment.reviewReason}</p>
		</section>
	{/if}

	<AdminDetailSection title="Payment attempt">
		<AdminDetailRow label="Provider">{payment.provider}</AdminDetailRow>
		<AdminDetailRow label="Method">Virtual Account · {payment.bank}</AdminDetailRow>
		<AdminDetailRow label="Provider reference">{payment.providerReference}</AdminDetailRow>
		<AdminDetailRow label="Amount">{formatIDR(payment.total)}</AdminDetailRow>
		<AdminDetailRow label="Created at"
			><time datetime={payment.createdAt}>{formatWitaDateTime(payment.createdAt)} WITA</time
			></AdminDetailRow
		>
		<AdminDetailRow label="Last updated"
			><time datetime={payment.updatedAt}>{formatWitaDateTime(payment.updatedAt)} WITA</time
			></AdminDetailRow
		>
	</AdminDetailSection>

	<AdminDetailSection title="Related booking">
		<AdminDetailRow label="Booking">
			<Link
				href={resolve('/admin/(staff)/bookings/[id]', { id: record.booking.id })}
				variant="plain"
				class="text-primary underline">{record.booking.id}</Link
			>
		</AdminDetailRow>
		<AdminDetailRow label="Booking status"
			><TransactionStatus value={{ booking: record.booking.status }} /></AdminDetailRow
		>
	</AdminDetailSection>

	<TransactionTimeline title="Payment timeline" events={payment.timeline} />
{:else}
	<h1 class="mt-2 text-3xl font-bold tracking-tight">Payment not found</h1>
	<p class="mt-2 text-sm text-ink-muted">Return to Payments to choose an available attempt.</p>
{/if}
