<script lang="ts">
	let {
		make,
		model,
		image,
		priority = false,
		thumbnail = false
	}: {
		make: string;
		model: string;
		image: string | null;
		priority?: boolean;
		thumbnail?: boolean;
	} = $props();
	let failedImage = $state<string | null>(null);
</script>

<div class="hero-artwork absolute inset-0 overflow-hidden" aria-hidden="true">
	<p
		class={[
			'hero-model absolute inset-x-0 text-center font-display font-bold',
			thumbnail ? 'text-ink-muted' : 'text-primary/10',
			model.length > 6 && 'hero-model-wide'
		]}
	>
		<span class="block">{make}</span>
		<span class="block">{model}</span>
	</p>
	{#if image && image !== failedImage}
		<img
			src={image}
			alt=""
			fetchpriority={priority ? 'high' : 'auto'}
			class="hero-car absolute left-1/2 max-w-none -translate-x-1/2 object-contain"
			draggable="false"
			onerror={() => (failedImage = image)}
		/>
	{/if}
</div>
