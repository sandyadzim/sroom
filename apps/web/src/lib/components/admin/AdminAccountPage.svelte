<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft, Pencil } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { isAxiosError } from 'axios';
	import { getApiError } from '#lib/api/staff.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import { adminAccountQuery } from '#lib/queries/identity.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';
	import AdminAccountForm from './AdminAccountForm.svelte';

	let { id, mode }: { id: string; mode: 'detail' | 'edit' } = $props();
	const isEdit = $derived(mode === 'edit');
	const account = createQuery(() => adminAccountQuery(id));
	const errorStatus = $derived(
		isAxiosError(account.error) ? account.error.response?.status : undefined
	);
	const notFound = $derived(errorStatus === 404 || errorStatus === 400);
	const forbidden = $derived(errorStatus === 403);
	const backToDetail = $derived(isEdit && !notFound && !forbidden);
	const backHref = $derived(
		backToDetail ? resolve('/admin/(staff)/admins/[id]', { id }) : resolve('admin/admins')
	);
	const title = $derived(isEdit ? 'Edit admin' : 'Admin details');
</script>

<svelte:head>
	<title>{title} — SROOM</title>
</svelte:head>

<Link href={backHref} class="gap-2 text-sm font-medium text-ink-muted hover:text-ink">
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" />
	{backToDetail ? 'Back to admin details' : 'Back to admin accounts'}
</Link>
<div class="mt-2 flex flex-wrap items-center justify-between gap-3">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">{title}</h1>
	{#if !isEdit && account.isSuccess}
		<Button href={resolve('/admin/(staff)/admins/[id]/edit', { id })}>
			<Pencil size={16} strokeWidth={1.5} aria-hidden="true" />
			Edit admin
		</Button>
	{/if}
</div>

{#if account.isPending}
	<p class="mt-6 border-t border-line pt-6 text-ink-muted" role="status">Loading admin account…</p>
{:else if account.isError}
	<div class="mt-6 border-t border-line pt-6">
		<p role="alert" class="text-negative">
			{#if notFound}
				Admin account not found.
			{:else if forbidden}
				Access denied. This page is for the owner.
			{:else}
				{getApiError(account.error)?.message ?? 'Could not load this admin account.'}
			{/if}
		</p>
		{#if !notFound && !forbidden && errorStatus !== 401}
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
	{@const admin = account.data}
	{#if isEdit}
		{#key admin.id}
			<AdminAccountForm mode="edit" account={admin} />
		{/key}
	{:else}
		<section
			aria-labelledby="account-details"
			class="mt-6 grid gap-4 border-t border-line pt-6 lg:grid-cols-[minmax(12rem,1fr)_minmax(0,2fr)] lg:gap-8"
		>
			<h2 id="account-details" class="text-lg/7 font-semibold">Account details</h2>
			<dl class="min-w-0 divide-y divide-line rounded-xl bg-page px-4 sm:px-5">
				{@render detailRow('Full name', admin.full_name.trim() || '—')}
				{@render detailRow('Email', admin.email)}
				{@render detailRow('Role', admin.role)}
				{@render detailRow('Status', admin.status === 'active' ? 'Active' : 'Inactive', {
					tone: admin.status === 'active' ? 'success' : 'danger'
				})}
				{@render detailRow('Account ID', admin.id)}
				{@render detailRow('Created at', formatWitaDateTime(admin.created_at), {
					datetime: admin.created_at
				})}
				{@render detailRow('Last updated', formatWitaDateTime(admin.updated_at), {
					datetime: admin.updated_at
				})}
			</dl>
		</section>
	{/if}
{/if}

{#snippet detailRow(
	label: string,
	value: string,
	options?: { tone?: 'success' | 'danger'; datetime?: string }
)}
	<div class="grid gap-1 py-4 text-sm/6 sm:grid-cols-[8rem_minmax(0,1fr)] sm:gap-4">
		<dt class="text-ink-muted">{label}</dt>
		<dd class="font-medium wrap-anywhere">
			{#if options?.tone}
				<StatusBadge label={value} tone={options.tone} />
			{:else if options?.datetime}
				<time datetime={options.datetime} class="tabular-nums">{value}</time>
			{:else}
				{value}
			{/if}
		</dd>
	</div>
{/snippet}
