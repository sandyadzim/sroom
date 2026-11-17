<script lang="ts">
	import { setRegistrationEmail } from '#lib/components/auth/registration-email.ts';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	setRegistrationEmail({ email: '' });
</script>

{@render children()}
