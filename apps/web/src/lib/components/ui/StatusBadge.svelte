<script lang="ts">
	let { label, tone }: { label: string; tone: 'success' | 'danger' } = $props();

	const tones = {
		success: 'bg-positive/10 text-positive',
		danger: 'bg-negative/10 text-negative'
	};
</script>

<span class={['inline-flex rounded-full px-2.5 py-1 text-xs font-semibold', tones[tone]]}>
	{label}
</span>
