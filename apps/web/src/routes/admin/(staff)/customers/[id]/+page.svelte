<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { isAxiosError } from 'axios';
	import { fetchCustomer } from '#lib/api/identity/customer.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import CustomerStatusForm from '#lib/components/admin/CustomerStatusForm.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import type { PageProps } from './$types';

	let { params }: PageProps = $props();
	const account = createQuery(() => {
		const id = params.id;
		return {
			queryKey: ['customer', id],
			queryFn: ({ signal }) => fetchCustomer(id, signal)
		};
	});
	const errorStatus = $derived(
		isAxiosError(account.error) ? account.error.response?.status : undefined
	);
</script>

<svelte:head>
	<title>Customer details — SROOM</title>
</svelte:head>

<Link
	href={resolve('/admin/(staff)/customers')}
	class="gap-2 text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" />
	Back to customers
</Link>
<h1 class="mt-2 text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Customer details</h1>

{#if account.isPending}
	<p class="mt-6 border-t border-line pt-6 text-ink-muted" role="status">Loading customer…</p>
{:else if account.isError}
	<div class="mt-6 border-t border-line pt-6">
		<p role="alert" class="text-negative">
			{#if errorStatus === 404 || errorStatus === 400}
				Customer not found.
			{:else if errorStatus === 403}
				Access denied.
			{:else}
				{getApiError(account.error)?.message ?? 'Could not load this customer.'}
			{/if}
		</p>
		{#if !errorStatus || errorStatus >= 500}
			<Button
				class="mt-3"
				variant="secondary"
				disabled={account.isFetching}
				onclick={() => account.refetch()}
			>
				{account.isFetching ? 'Loading…' : 'Try again'}
			</Button>
		{/if}
	</div>
{:else if account.isSuccess}
	{@const customer = account.data}
	<section
		aria-labelledby="customer-account-details"
		class="mt-6 grid gap-4 border-t border-line pt-6 lg:grid-cols-[minmax(12rem,1fr)_minmax(0,2fr)] lg:gap-8"
	>
		<h2 id="customer-account-details" class="text-lg/7 font-semibold">Account details</h2>
		<dl class="min-w-0 divide-y divide-line rounded-xl bg-page px-4 sm:px-5">
			{@render detailRow('Full name', customer.full_name.trim() || '—')}
			{@render detailRow('Email', customer.email)}
			{@render detailRow('Account ID', customer.id)}
			{@render detailRow(
				'Registered at',
				formatWitaDateTime(customer.created_at),
				customer.created_at
			)}
			{@render detailRow(
				'Last updated',
				formatWitaDateTime(customer.updated_at),
				customer.updated_at
			)}
		</dl>
	</section>

	{#key customer.id}
		<CustomerStatusForm {customer} />
	{/key}
{/if}

{#snippet detailRow(label: string, value: string, datetime?: string)}
	<div class="grid gap-1 py-4 text-sm/6 sm:grid-cols-[8rem_minmax(0,1fr)] sm:gap-4">
		<dt class="text-ink-muted">{label}</dt>
		<dd class="font-medium wrap-anywhere">
			{#if datetime}<time {datetime} class="tabular-nums">{value}</time>{:else}{value}{/if}
		</dd>
	</div>
{/snippet}
