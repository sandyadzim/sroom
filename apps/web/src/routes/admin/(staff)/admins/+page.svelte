<script lang="ts">
	import { adminAccountsEndpoint } from '#lib/api/identity/admin.ts';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import DataTableGenerator from '#lib/components/ui/data-table/DataTableGenerator.svelte';
	import type { DataTableConfig } from '#lib/types/data-table.ts';
	import type { AdminAccount } from '#lib/types/identity.ts';
	import { formatWitaDateTime } from '#lib/utils/date.ts';

	const config: DataTableConfig<AdminAccount> = {
		endpoint: adminAccountsEndpoint,
		label: 'Admin accounts',
		rowKey: (row) => row.id,
		columns: [
			{ label: 'Full name', cell: fullName },
			{ label: 'Email', field: 'email' },
			{ label: 'Status', cell: accountStatus },
			{ label: 'Created at', cell: createdAt }
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
		emptyMessage: 'No admin accounts yet.',
		filteredEmptyMessage: 'No accounts match your filters.'
	};
</script>

<svelte:head>
	<title>Admin accounts — SROOM</title>
</svelte:head>

<h1 class="mb-7 text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Admin accounts</h1>

<DataTableGenerator {config} />

{#snippet fullName(row: AdminAccount)}
	<span class="font-medium">{row.full_name.trim() || '—'}</span>
{/snippet}

{#snippet accountStatus(row: AdminAccount)}
	{@const active = row.status === 'active'}
	<StatusBadge label={active ? 'Active' : 'Inactive'} tone={active ? 'success' : 'danger'} />
{/snippet}

{#snippet createdAt(row: AdminAccount)}
	<time datetime={row.created_at} class="whitespace-nowrap tabular-nums">
		{formatWitaDateTime(row.created_at)}
	</time>
{/snippet}
