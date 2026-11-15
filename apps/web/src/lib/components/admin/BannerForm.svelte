<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { ImagePlus, LoaderCircle } from '@lucide/svelte';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { onDestroy, tick, untrack } from 'svelte';
	import { saveBannerSlot } from '#lib/api/catalog/banner.ts';
	import { uploadCatalogImage } from '#lib/api/catalog/image.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import ImageWithPlaceholder from '#lib/components/ui/ImageWithPlaceholder.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { bannerSlotQuery, bannerSlotsQuery } from '#lib/queries/banner.ts';
	import { brandOptionsQuery } from '#lib/queries/brand.ts';
	import type { BannerInput, BannerSlot } from '#lib/types/catalog.ts';
	import { useToast } from '#lib/ui/toast.ts';

	let { slot }: { slot: BannerSlot } = $props();
	const original = untrack(() => slot.content);
	const formId = $props.id();
	const queryClient = useQueryClient();
	const toast = useToast();
	const brands = createQuery(() => brandOptionsQuery);
	const brandOptions = $derived(
		brands.data?.map((brand) => ({ value: brand.id, label: brand.name })) ?? []
	);
	const imageTypes = ['image/jpeg', 'image/png', 'image/webp'];
	const listHref = $derived(resolve('/admin/(staff)/banners'));

	let values = $state<BannerInput>({
		brand_id: original?.brand_id ?? '',
		car_model: original?.car_model ?? '',
		car_slug: original?.car_slug ?? '',
		headline_line_1: original?.headline_line_1 ?? '',
		headline_line_2: original?.headline_line_2 ?? '',
		description: original?.description ?? '',
		image_object_key: original?.image_object_key ?? ''
	});
	let errors = $state<Partial<Record<keyof BannerInput, string>>>({});
	let fileInput = $state<HTMLInputElement>();
	let upload = $state.raw<AbortController>();
	let uploaded = $state.raw<File>();
	let previewUrl = $state<string>();
	const save = createMutation(() => ({
		mutationFn: (input: BannerInput) => saveBannerSlot(slot.position, input)
	}));
	const imageUrl = $derived(uploaded ? previewUrl : original?.image_url);
	const imageName = $derived(uploaded?.name ?? original?.image_object_key.split('/').at(-1));
	const fields = [
		{ name: 'car_model', label: 'Car model', limit: 100 },
		{
			name: 'car_slug',
			label: 'Car slug',
			limit: 150,
			help: 'Use the slug from the car detail URL, for example fortuner-vrz-at.'
		},
		{ name: 'headline_line_1', label: 'Headline line 1', limit: 100 },
		{ name: 'headline_line_2', label: 'Headline line 2', limit: 100 }
	] as const;

	onDestroy(() => upload?.abort());

	$effect(() => {
		if (!uploaded) return;
		const url = URL.createObjectURL(uploaded);
		previewUrl = url;
		return () => URL.revokeObjectURL(url);
	});

	function validate(name: keyof BannerInput) {
		const value = values[name].trim();
		const field = fields.find((item) => item.name === name);
		if (!value) {
			if (name === 'brand_id') return 'Select a brand.';
			if (name === 'image_object_key') return 'Upload a banner image.';
			return `Enter the ${field?.label.toLowerCase() ?? 'description'}.`;
		}
		const limit = field?.limit ?? (name === 'description' ? 500 : undefined);
		if (limit && Array.from(value).length > limit) return `Use at most ${limit} characters.`;
		if (name === 'car_slug' && !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(value)) {
			return 'Use lowercase letters, numbers, and single hyphens between words.';
		}
	}

	async function selectFile(input: HTMLInputElement) {
		const file = input.files?.[0];
		input.value = '';
		if (!file || upload || save.isPending) return;
		if (file.size > 5 * 1024 * 1024) {
			errors.image_object_key = 'Choose an image no larger than 5 MiB.';
			return;
		}
		if (!file.size || (file.type && !imageTypes.includes(file.type))) {
			errors.image_object_key = 'Choose a JPEG, PNG, or WebP image.';
			return;
		}

		errors.image_object_key = undefined;
		const controller = new AbortController();
		upload = controller;
		try {
			const key = await uploadCatalogImage(file, controller.signal);
			if (!controller.signal.aborted) {
				uploaded = file;
				values.image_object_key = key;
			}
		} catch (cause) {
			if (!controller.signal.aborted) {
				errors.image_object_key =
					getApiError(cause)?.message ?? 'Could not upload the image. Please try again.';
			}
		} finally {
			if (upload === controller) upload = undefined;
		}
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (upload || save.isPending || !brandOptions.length) return;
		const fieldOrder = [
			'brand_id',
			...fields.map((field) => field.name),
			'description',
			'image_object_key'
		] as const;
		for (const name of fieldOrder) {
			if (name === 'image_object_key') errors[name] ??= validate(name);
			else errors[name] = validate(name);
		}
		const invalid = fieldOrder.find((name) => errors[name]);
		if (invalid) {
			toast({ kind: 'warning', message: 'Check the highlighted fields and try again.' });
			await tick();
			document.getElementById(`${formId}-${invalid}`)?.focus();
			return;
		}

		try {
			const { data, message } = await save.mutateAsync({
				brand_id: values.brand_id,
				car_model: values.car_model.trim(),
				car_slug: values.car_slug.trim(),
				headline_line_1: values.headline_line_1.trim(),
				headline_line_2: values.headline_line_2.trim(),
				description: values.description.trim(),
				image_object_key: values.image_object_key
			});
			queryClient.setQueryData(bannerSlotQuery(data.position).queryKey, data);
			queryClient.setQueryData(bannerSlotsQuery.queryKey, (slots) =>
				slots?.map((current) => (current.position === data.position ? data : current))
			);
			void queryClient.invalidateQueries({ queryKey: bannerSlotsQuery.queryKey, exact: true });
			toast({ kind: 'success', message });
			await goto(listHref);
		} catch (cause) {
			const error = getApiError(cause);
			if (error?.code === 'BRAND_NOT_FOUND') errors.brand_id = error.message;
			toast({
				kind: 'error',
				message: error?.message ?? 'Could not save the banner. Please try again.'
			});
		}
	}
</script>

{#if brands.isError && !brandOptions.length}
	<div class="mt-5">
		<p role="alert" class="text-negative">
			{getApiError(brands.error)?.message ?? 'Could not load brands.'}
		</p>
		<Button
			class="mt-3"
			variant="secondary"
			disabled={brands.isFetching}
			onclick={() => brands.refetch()}>Try again</Button
		>
	</div>
{:else if brands.isSuccess && !brandOptions.length}
	<p class="mt-5 text-ink-muted">
		Add a brand before filling this slot.
		<Button href={resolve('/admin/(staff)/brands')} variant="ghost" class="px-3"
			>Go to Brands</Button
		>
	</p>
{/if}

<form
	method="post"
	class="mt-6 max-w-5xl border-t border-line pt-6"
	novalidate
	aria-busy={save.isPending || !!upload}
	onsubmit={submit}
>
	<div class="grid items-start gap-6 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
		<div class="space-y-5">
			<SelectField
				id={`${formId}-brand_id`}
				name="brand_id"
				label="Brand"
				placeholder={brands.isPending ? 'Loading brands…' : 'Select a brand'}
				options={brandOptions}
				bind:value={values.brand_id}
				error={errors.brand_id}
				disabled={save.isPending || !brandOptions.length}
				onchange={() => (errors.brand_id = validate('brand_id'))}
				required
			/>

			{#each fields as { name, label, limit, ...details } (name)}
				<TextField
					id={`${formId}-${name}`}
					{name}
					{label}
					help={'help' in details ? details.help : undefined}
					maxlength={limit}
					value={values[name]}
					error={errors[name]}
					disabled={save.isPending}
					oninput={(event) => {
						values[name] = event.currentTarget.value;
						if (errors[name]) errors[name] = validate(name);
					}}
					onblur={() => (errors[name] = validate(name))}
					required
				/>
			{/each}

			<div>
				<label for={`${formId}-description`} class="mb-2 block text-sm font-medium">
					Description<span class="ml-1 text-negative" aria-hidden="true">*</span>
				</label>
				<textarea
					id={`${formId}-description`}
					name="description"
					rows="3"
					maxlength="500"
					value={values.description}
					oninput={(event) => {
						values.description = event.currentTarget.value;
						if (errors.description) errors.description = validate('description');
					}}
					disabled={save.isPending}
					onblur={() => (errors.description = validate('description'))}
					aria-invalid={errors.description ? 'true' : undefined}
					aria-describedby={errors.description ? `${formId}-description-error` : undefined}
					class={[
						'w-full resize-y rounded-xl border bg-panel px-4 py-3 focus-visible:outline-offset-2',
						errors.description ? 'border-negative' : 'border-line hover:border-ink-muted'
					]}
					required></textarea>
				<div aria-live="polite">
					{#if errors.description}<p
							id={`${formId}-description-error`}
							class="mt-2 text-sm text-negative"
						>
							{errors.description}
						</p>{/if}
				</div>
			</div>
		</div>

		<div class="min-w-0 rounded-xl border border-line bg-page p-4">
			<h2 class="mb-3 text-sm font-semibold">
				Banner image <span class="text-negative" aria-hidden="true">*</span>
			</h2>
			{#if imageName}
				<ImageWithPlaceholder
					src={imageUrl}
					alt="Banner image preview"
					class="aspect-video w-full rounded-lg bg-panel-muted"
				/>
				<p class="mt-2 truncate text-sm text-ink-muted" title={imageName}>{imageName}</p>
			{/if}
			<input
				bind:this={fileInput}
				type="file"
				name="image"
				accept={imageTypes.join(',')}
				aria-label="Upload banner image"
				tabindex="-1"
				class="sr-only"
				disabled={save.isPending || !!upload}
				onchange={(event) => selectFile(event.currentTarget)}
			/>
			<Button
				id={`${formId}-image_object_key`}
				variant="ghost"
				size="custom"
				class={[
					'w-full rounded-lg border border-dashed px-3 py-3',
					imageName && 'mt-3',
					errors.image_object_key ? 'border-negative' : 'border-line'
				]}
				aria-invalid={errors.image_object_key ? 'true' : undefined}
				aria-describedby={errors.image_object_key
					? `${formId}-image-help ${formId}-image-error`
					: `${formId}-image-help`}
				disabled={save.isPending || !!upload}
				onclick={() => fileInput?.click()}
			>
				{#if upload}
					<LoaderCircle
						size={18}
						strokeWidth={1.5}
						aria-hidden="true"
						class="motion-safe:animate-spin"
					/> Uploading…
				{:else}
					<ImagePlus size={18} strokeWidth={1.5} aria-hidden="true" />
					{imageName ? 'Replace image' : 'Add image'}
				{/if}
			</Button>
			<p id={`${formId}-image-help`} class="mt-2 text-sm text-ink-muted">
				JPEG, PNG, or WebP up to 5 MiB.
			</p>
			<div aria-live="polite">
				<span class="sr-only"
					>{upload ? 'Uploading image…' : uploaded ? 'Image uploaded.' : ''}</span
				>
				{#if errors.image_object_key}
					<p id={`${formId}-image-error`} class="mt-2 text-sm text-negative">
						{errors.image_object_key}
					</p>
					{#if values.image_object_key}
						<Button
							size="custom"
							variant="ghost"
							class="rounded-lg px-2"
							onclick={() => (errors.image_object_key = undefined)}>Keep current image</Button
						>
					{/if}
				{/if}
			</div>
		</div>
	</div>

	<div class="mt-6 flex flex-wrap justify-end gap-2 border-t border-line pt-4">
		<Button href={listHref} variant="secondary">Cancel</Button>
		<Button type="submit" disabled={save.isPending || !!upload || !brandOptions.length}>
			{save.isPending ? 'Saving…' : 'Save banner'}
		</Button>
	</div>
</form>
