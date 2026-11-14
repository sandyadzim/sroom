<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';
	import AdminAccountForm from '#lib/components/admin/AdminAccountForm.svelte';
	import Link from '#lib/components/ui/Link.svelte';
</script>

<svelte:head>
	<title>Create admin — SROOM</title>
</svelte:head>

<Link
	href={resolve('admin/admins')}
	class="gap-2 text-sm font-medium text-ink-muted hover:text-ink"
>
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" />
	Back to admin accounts
</Link>
<h1 class="mt-2 text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Create admin</h1>

<AdminAccountForm mode="create" />
