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
		image: string;
		priority?: boolean;
		thumbnail?: boolean;
	} = $props();
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
	<img
		src={image}
		alt=""
		width="1536"
		height="1024"
		fetchpriority={priority ? 'high' : 'auto'}
		class="hero-car absolute left-1/2 max-w-none -translate-x-1/2 object-contain"
		draggable="false"
	/>
</div>
