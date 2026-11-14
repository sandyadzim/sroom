<script lang="ts">
	import { resolve } from '$app/paths';
	import { Plus } from '@lucide/svelte';
	import { adminAccountsEndpoint } from '#lib/api/identity/admin.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Link from '#lib/components/ui/Link.svelte';
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
			{ label: 'Created at', cell: createdAt },
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
		emptyMessage: 'No admin accounts yet.',
		filteredEmptyMessage: 'No accounts match your filters.'
	};
</script>

<svelte:head>
	<title>Admin accounts — SROOM</title>
</svelte:head>

<div class="mb-7 flex flex-wrap items-center justify-between gap-3">
	<h1 class="text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Admin accounts</h1>
	<Button href={resolve('admin/admins/new')}>
		<Plus size={18} strokeWidth={1.5} aria-hidden="true" />
		Create admin
	</Button>
</div>

<DataTableGenerator {config} />

{#snippet fullName(row: AdminAccount)}
	<Link
		href={resolve('/admin/(staff)/admins/[id]', { id: row.id })}
		variant="plain"
		class="font-medium text-primary underline decoration-primary/30 underline-offset-4 hover:decoration-primary"
	>
		{row.full_name.trim() || row.email}
	</Link>
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

{#snippet actions(row: AdminAccount)}
	<Button
		href={resolve('/admin/(staff)/admins/[id]', { id: row.id })}
		variant="ghost"
		size="custom"
		class="rounded-lg px-3"
		aria-label={`View ${row.full_name.trim() || row.email}`}
	>
		View
	</Button>
{/snippet}
