<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { customersEndpoint, updateCustomerStatus } from '#lib/api/identity/customer.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import AdminFormActions from '#lib/components/admin/AdminFormActions.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import StatusBadge from '#lib/components/ui/StatusBadge.svelte';
	import type { CustomerAccount } from '#lib/types/identity.ts';
	import { useToast } from '#lib/ui/toast.ts';

	let { customer }: { customer: CustomerAccount } = $props();
	const queryClient = useQueryClient();
	const showToast = useToast();
	let selectedStatus = $state<CustomerAccount['status']>();
	const active = $derived(customer.status === 'active');
	const options = [
		{ value: 'active', label: 'Active' },
		{ value: 'inactive', label: 'Inactive' }
	];
	const saveStatus = createMutation(() => ({
		mutationFn: ({ id, status }: { id: string; status: CustomerAccount['status'] }) =>
			updateCustomerStatus(id, status),
		onSuccess: async (updatedCustomer) => {
			const queryKey = ['customer', updatedCustomer.id];
			await queryClient.cancelQueries({ queryKey, exact: true });
			queryClient.setQueryData(queryKey, updatedCustomer);
			await queryClient.invalidateQueries({
				queryKey: [customersEndpoint],
				refetchType: 'none'
			});
		}
	}));

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedStatus || selectedStatus === customer.status || saveStatus.isPending) return;
		try {
			await saveStatus.mutateAsync({ id: customer.id, status: selectedStatus });
		} catch (cause) {
			showToast({
				kind: 'error',
				message: getApiError(cause)?.message ?? 'Could not save the status. Please try again.'
			});
			return;
		}

		showToast({ kind: 'success', message: 'Customer status updated.' });
		await goto(resolve('/admin/(staff)/customers'));
	}
</script>

<section
	aria-labelledby="customer-account-status"
	class="mt-6 grid gap-4 border-t border-line pt-6 lg:grid-cols-[minmax(12rem,1fr)_minmax(0,2fr)] lg:gap-8"
>
	<div class="flex flex-wrap items-start gap-2">
		<h2 id="customer-account-status" class="text-lg/7 font-semibold">Sign-in status</h2>
		<StatusBadge label={active ? 'Active' : 'Inactive'} tone={active ? 'success' : 'danger'} />
	</div>
	<form method="post" onsubmit={submit} aria-busy={saveStatus.isPending} class="max-w-sm">
		<SelectField
			name="status"
			label="New status"
			{options}
			value={selectedStatus ?? customer.status}
			help="Inactive customers cannot sign in again. Existing sessions stay active until sign-out or expiry."
			disabled={saveStatus.isPending}
			onchange={(value) => {
				selectedStatus = value === 'inactive' ? 'inactive' : 'active';
			}}
		/>
		<AdminFormActions>
			<Button
				type="submit"
				disabled={!selectedStatus || selectedStatus === customer.status || saveStatus.isPending}
			>
				{saveStatus.isPending ? 'Saving…' : 'Save status'}
			</Button>
		</AdminFormActions>
	</form>
</section>
