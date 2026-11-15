<script lang="ts">
	import { Dialog as BitsDialog } from 'bits-ui';
	import { X } from '@lucide/svelte';
	import type { Snippet } from 'svelte';
	import Button from './Button.svelte';

	let {
		title,
		description,
		placement = 'center',
		size = 'default',
		closeLabel = 'Close dialog',
		onkeydown,
		trigger,
		children,
		open = $bindable(false)
	}: {
		title: string;
		description: string;
		placement?: 'center' | 'left';
		size?: 'default' | 'compact';
		closeLabel?: string;
		onkeydown?: BitsDialog.ContentProps['onkeydown'];
		trigger: Snippet<[{ props: BitsDialog.TriggerProps }]>;
		children: Snippet;
		open?: boolean;
	} = $props();
</script>

<BitsDialog.Root bind:open>
	<BitsDialog.Trigger>
		{#snippet child({ props })}{@render trigger({ props })}{/snippet}
	</BitsDialog.Trigger>
	<BitsDialog.Portal>
		<BitsDialog.Overlay class="dialog-overlay fixed inset-0 z-60 bg-ink/50" />
		<BitsDialog.Content
			{onkeydown}
			class={[
				'dialog-content fixed z-60 overflow-y-auto overscroll-contain bg-panel p-4 sm:p-6',
				placement === 'left'
					? 'inset-y-0 left-0 flex w-72 max-w-[calc(100%-2rem)] flex-col'
					: [
							'top-1/2 left-1/2 max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] -translate-x-1/2 -translate-y-1/2 rounded-panel',
							size === 'compact' ? 'max-w-lg' : 'max-w-5xl'
						]
			]}
		>
			<div class="flex items-center justify-between gap-4">
				<BitsDialog.Title class="min-w-0 text-lg/7 font-semibold wrap-anywhere"
					>{title}</BitsDialog.Title
				>
				<BitsDialog.Close>
					{#snippet child({ props })}
						<Button {...props} variant="ghost" size="icon" aria-label={closeLabel}>
							<X size={20} strokeWidth={1.5} aria-hidden="true" />
						</Button>
					{/snippet}
				</BitsDialog.Close>
			</div>
			<BitsDialog.Description class="sr-only">{description}</BitsDialog.Description>
			<div class={['mt-4', placement === 'left' && 'flex flex-1 flex-col']}>
				{@render children()}
			</div>
		</BitsDialog.Content>
	</BitsDialog.Portal>
</BitsDialog.Root>
