<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';
	import { changePassword } from '#lib/api/identity/profile.ts';
	import ChangePasswordForm from '#lib/components/auth/ChangePasswordForm.svelte';
	import Link from '#lib/components/ui/Link.svelte';

	const profileHref = $derived(resolve('admin/profile'));
</script>

<svelte:head>
	<title>Change password — SROOM</title>
</svelte:head>

<Link href={profileHref} class="gap-2 text-sm font-medium text-ink-muted hover:text-ink">
	<ArrowLeft size={16} strokeWidth={1.5} aria-hidden="true" />
	Back to profile
</Link>
<h1 class="mt-2 text-3xl/10 font-bold tracking-tight sm:text-4xl/12">Change password</h1>

<ChangePasswordForm profileRoute="/admin/(staff)/profile" {changePassword} />
