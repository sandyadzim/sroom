<script lang="ts">
	import { CircleAlert, CircleCheck, TriangleAlert, X } from '@lucide/svelte';
	import type { ToastFeedback } from '#lib/ui/toast.ts';
	import Button from './Button.svelte';

	let {
		feedback,
		paused,
		onDismiss
	}: { feedback: ToastFeedback; paused: boolean; onDismiss: () => void } = $props();
	const styles = {
		success: { icon: CircleCheck, color: 'text-positive', label: 'Success' },
		error: { icon: CircleAlert, color: 'text-negative', label: 'Error' },
		warning: { icon: TriangleAlert, color: 'text-caution', label: 'Warning' }
	};
	const style = $derived(styles[feedback.kind]);
	let remaining = 4000;

	$effect(() => {
		if (paused) return;
		const started = performance.now();
		const timer = setTimeout(() => onDismiss(), remaining);
		return () => {
			clearTimeout(timer);
			remaining = Math.max(0, remaining - (performance.now() - started));
		};
	});
</script>

<div
	class="flex min-h-14 items-center gap-3 rounded-xl bg-panel py-1.5 pr-1.5 pl-4 text-ink shadow-[0_4px_18px_rgba(16,24,32,0.12)] ring-1 ring-ink/10"
>
	<style.icon size={18} strokeWidth={1.75} class={['shrink-0', style.color]} aria-hidden="true" />
	<p class="min-w-0 flex-1 py-2 text-sm/5 font-medium wrap-anywhere">
		<span class="sr-only">{style.label}. </span>{feedback.message}
	</p>
	<Button variant="ghost" size="icon" aria-label="Dismiss notification" onclick={onDismiss}>
		<X size={16} strokeWidth={1.75} aria-hidden="true" />
	</Button>
</div>
