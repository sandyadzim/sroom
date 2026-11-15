<script lang="ts">
	import { resolve } from '$app/paths';
	import { customersEndpoint } from '#lib/api/identity/customer.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import DataTableGenerator from '#lib/components/ui/data-table/DataTableGenerator.svelte';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import type { CustomerAccount } from '#lib/types/identity.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	const config: DataTableConfig<CustomerAccount> = {
		endpoint: customersEndpoint,
		label: 'Customers',
		rowKey: (row) => row.id,
		columns: [
			{ label: 'Customer', cell: customerIdentity },
			{ label: 'Status', cell: accountStatus },
			{ label: 'Registered at', cell: createdAt },
			{ label: 'Actions', cell: actions, sticky: 'right' }
		],
		filters: [
			{ name: 'search', label: 'Search', type: 'search', placeholder: 'Search by name or email' },
			{
				name: 'status',
				label: 'Status',
				type: 'select',
				placeholder: 'Select status',
				options: [
					{ value: 'active', label: 'Active' },
					{ value: 'inactive', label: 'Inactive' }
				]
			}
		],
		pageSize: 20,
		emptyMessage: 'No customers have registered yet.',
		filteredEmptyMessage: 'No customers match your filters.'
	};
</script>

<svelte:head>
	<title>Customers — SROOM</title>
</svelte:head>

<h1 class="mb-7 text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Customers</h1>

<DataTableGenerator {config} />

{#snippet customerIdentity(row: CustomerAccount)}
	<Link
		href={resolve('/admin/(staff)/customers/[id]', { id: row.id })}
		variant="plain"
		class="font-medium text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
	>
		{row.full_name.trim() || row.email}
	</Link>
	{#if row.full_name.trim()}
		<span class="mt-1 block text-ink-muted">{row.email}</span>
	{/if}
{/snippet}

{#snippet accountStatus(row: CustomerAccount)}
	{@const active = row.status === 'active'}
	<StatusBadge label={active ? 'Active' : 'Inactive'} tone={active ? 'success' : 'danger'} />
{/snippet}

{#snippet createdAt(row: CustomerAccount)}
	<time datetime={row.created_at} class="whitespace-nowrap tabular-nums">
		{formatWitaDateTime(row.created_at)}
	</time>
{/snippet}

{#snippet actions(row: CustomerAccount)}
	<Button
		href={resolve('/admin/(staff)/customers/[id]', { id: row.id })}
		variant="ghost"
		size="custom"
		class="rounded-lg px-3"
		aria-label={`View ${row.full_name.trim() || row.email}`}
	>
		View
	</Button>
{/snippet}
