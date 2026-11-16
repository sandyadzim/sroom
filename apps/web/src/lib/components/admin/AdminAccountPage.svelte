<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft, Pencil } from '@lucide/svelte';
	import { createQuery } from '@tanstack/svelte-query';
	import { isAxiosError } from 'axios';
	import { getApiError } from '#lib/api/staff.ts';
	import AdminDetailRow from '#lib/components/admin/AdminDetailRow.svelte';
	import AdminDetailSection from '#lib/components/admin/AdminDetailSection.svelte';
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
		<AdminDetailSection title="Account details">
			<AdminDetailRow label="Full name">{admin.full_name.trim() || '—'}</AdminDetailRow>
			<AdminDetailRow label="Email">{admin.email}</AdminDetailRow>
			<AdminDetailRow label="Role">{admin.role}</AdminDetailRow>
			<AdminDetailRow label="Status">
				<StatusBadge
					label={admin.status === 'active' ? 'Active' : 'Inactive'}
					tone={admin.status === 'active' ? 'success' : 'danger'}
				/>
			</AdminDetailRow>
			<AdminDetailRow label="Account ID">{admin.id}</AdminDetailRow>
			<AdminDetailRow label="Created at">
				<time datetime={admin.created_at} class="tabular-nums"
					>{formatWitaDateTime(admin.created_at)}</time
				>
			</AdminDetailRow>
			<AdminDetailRow label="Last updated">
				<time datetime={admin.updated_at} class="tabular-nums"
					>{formatWitaDateTime(admin.updated_at)}</time
				>
			</AdminDetailRow>
		</AdminDetailSection>
	{/if}
{/if}
