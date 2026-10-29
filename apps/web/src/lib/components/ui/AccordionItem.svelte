<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Plus } from '@lucide/svelte';
	import Button from './Button.svelte';

	let { question, children }: { question: string; children: Snippet } = $props();
	const id = $props.id();
	let open = $state(false);
</script>

<div class="rounded-card bg-panel-muted">
	<h3>
		<Button
			id={`${id}-trigger`}
			variant="ghost"
			size="custom"
			class="w-full rounded-card px-5 py-5 text-left sm:px-6"
			aria-expanded={open}
			aria-controls={`${id}-content`}
			onclick={() => (open = !open)}
		>
			<span class="flex w-full items-center justify-between gap-5">
				<span class="font-display text-base/6 font-semibold sm:text-lg/7">{question}</span>
				<span class={['accordion-icon inline-flex shrink-0', open && 'rotate-45']}>
					<Plus size={20} strokeWidth={1.5} aria-hidden="true" />
				</span>
			</span>
		</Button>
	</h3>
	<div
		id={`${id}-content`}
		class="accordion-content"
		data-open={open}
		aria-hidden={!open}
		inert={!open}
	>
		<div class="min-h-0 overflow-hidden">
			<div class="px-5 pb-6 text-base/7 text-ink-muted sm:px-6 sm:pr-16">
				{@render children()}
			</div>
		</div>
	</div>
</div>

<style>
	.accordion-content {
		display: grid;
		grid-template-rows: 0fr;
		visibility: hidden;
		transition:
			grid-template-rows 240ms cubic-bezier(0.23, 1, 0.32, 1),
			visibility 0s 240ms;
	}

	.accordion-content[data-open='true'] {
		grid-template-rows: 1fr;
		visibility: visible;
		transition-delay: 0s;
	}

	.accordion-icon {
		transition: rotate 200ms cubic-bezier(0.23, 1, 0.32, 1);
	}

	@media (prefers-reduced-motion: reduce) {
		.accordion-content,
		.accordion-icon {
			transition: none;
		}
	}
</style>
