<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { Check, ImagePlus } from '@lucide/svelte';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { onDestroy, tick, untrack } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import { carsEndpoint, createCar, updateCar } from '#lib/api/catalog/car.ts';
	import { uploadCatalogImage } from '#lib/api/catalog/image.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import AdminFormActions from '#lib/components/admin/AdminFormActions.svelte';
	import Button from '#lib/components/ui/Button.svelte';
	import NumberField from '#lib/components/ui/NumberField.svelte';
	import SelectField from '#lib/components/ui/SelectField.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import { brandOptionsQuery } from '#lib/queries/brand.ts';
	import { carDetailQuery } from '#lib/queries/car.ts';
	import type {
		AdminCarDetail,
		AdminCarImage,
		CarFields,
		CarPhotoInput
	} from '#lib/types/catalog.ts';
	import { useToast } from '#lib/ui/toast.ts';
	import {
		carBodyTypes,
		carCategories,
		carFuelTypes,
		carPhotoInput,
		carStatuses,
		carTransmissions,
		changedCarFields,
		suggestCarPhotoMetadata
	} from '#lib/utils/car.ts';
	import CarPhotoFields from './CarPhotoFields.svelte';

	type Photo = CarPhotoInput & { id: string; url: string | null };
	type Values = Omit<CarFields, 'seats' | 'daily_rate_idr'> & {
		seats: string;
		daily_rate_idr: string;
	};
	type Field = keyof Values;
	type PhotoKind = AdminCarImage['kind'];

	let { car }: { car?: AdminCarDetail } = $props();
	const ids = $props.id();
	const original = untrack(() => car);
	const toast = useToast();
	const queryClient = useQueryClient();
	const brands = createQuery(() => brandOptionsQuery);
	const brandOptions = $derived.by(() => {
		const options =
			brands.data
				?.filter((brand) => brand.status === 'active')
				.map((brand) => ({ value: brand.id, label: brand.name })) ?? [];
		if (original && !options.some((option) => option.value === original.brand_id)) {
			options.unshift({ value: original.brand_id, label: original.brand_name });
		}
		return options;
	});

	let form: HTMLFormElement;
	let stepHeading: HTMLHeadingElement;
	let coverInput = $state<HTMLInputElement>();
	let galleryInput = $state<HTMLInputElement>();
	let values = $state<Values>({
		brand_id: original?.brand_id ?? '',
		model: original?.model ?? '',
		trim: original?.trim ?? '',
		category: original?.category ?? '',
		body_type: original?.body_type ?? '',
		seats: original ? String(original.seats) : '',
		transmission: original?.transmission ?? '',
		fuel_type: original?.fuel_type ?? '',
		description: original?.description ?? '',
		daily_rate_idr: original ? String(original.daily_rate_idr) : '',
		status: original?.status ?? 'active'
	});
	function draftPhoto(image: AdminCarImage): Photo {
		return { ...carPhotoInput(image), id: image.id, url: image.url };
	}
	const savedCover = original?.images.find((photo) => photo.kind === 'cover');
	let cover = $state(savedCover && draftPhoto(savedCover));
	let gallery = $state(
		original?.images.filter((photo) => photo.kind === 'gallery').map(draftPhoto) ?? []
	);
	const previewUrls = new SvelteSet<string>();
	let upload = $state.raw<AbortController>();
	let uploading = $state<PhotoKind>();
	let uploadErrors = $state<Partial<Record<PhotoKind, string>>>({});
	let touched = $state<Partial<Record<Field, boolean>>>({});
	let imageSubmitted = $state(false);
	let step = $state(0);

	const steps: readonly { label: string; fields: readonly Field[] }[] = [
		{ label: 'General information', fields: ['brand_id', 'model', 'trim', 'description'] },
		{
			label: 'Specifications',
			fields: ['category', 'body_type', 'seats', 'transmission', 'fuel_type']
		},
		{ label: 'Pricing', fields: ['daily_rate_idr', 'status'] },
		{ label: 'Images', fields: [] }
	];
	const imageStep = steps.length - 1;
	const currentStep = $derived(steps[step]);

	const fields = $derived<CarFields>({
		...values,
		model: values.model.trim(),
		trim: values.trim.trim(),
		description: values.description.trim(),
		seats: Number(values.seats),
		daily_rate_idr: Number(values.daily_rate_idr)
	});
	// A legacy record can be edited without adding photos unless an image rule is triggered.
	const needsCover = $derived(
		!original || (values.status === 'active' && original.status !== 'active')
	);
	const needsGallery = !original || original.images.some((photo) => photo.kind === 'gallery');
	const patch = $derived(
		original &&
			changedCarFields(original, fields, cover && carPhotoInput(cover), gallery.map(carPhotoInput))
	);
	const dirty = $derived(!patch || Object.keys(patch).length > 0);
	const save = createMutation(() => ({
		mutationFn: async () => {
			if (original && patch) return updateCar(original.id, patch);
			if (!cover) throw new Error('Upload a cover photo.');
			return createCar({
				...fields,
				cover: carPhotoInput(cover),
				gallery: gallery.map(carPhotoInput)
			});
		}
	}));
	const busy = $derived(save.isPending || !!upload);
	const blocked = $derived(
		busy || (step === 0 && !original && brandOptions.length === 0) || (step === imageStep && !dirty)
	);
	const cancelHref = $derived(
		original
			? resolve('/admin/(staff)/cars/[id]', { id: original.id })
			: resolve('/admin/(staff)/cars')
	);

	onDestroy(() => {
		upload?.abort();
		for (const url of previewUrls) URL.revokeObjectURL(url);
	});

	function validate(field: Field) {
		const value = values[field].trim();
		switch (field) {
			case 'brand_id':
				return value ? undefined : 'Select a brand.';
			case 'model':
				return value ? undefined : 'Enter a model.';
			case 'description':
				return value ? undefined : 'Enter a description.';
			case 'category':
				return value ? undefined : 'Select a category.';
			case 'body_type':
				return value ? undefined : 'Select a body type.';
			case 'transmission':
				return value ? undefined : 'Select a transmission.';
			case 'fuel_type':
				return value ? undefined : 'Select a fuel type.';
			case 'status':
				return value ? undefined : 'Select a status.';
			case 'seats':
				return /^\d+$/.test(value) && fields.seats > 0 && fields.seats <= 2147483647
					? undefined
					: 'Enter a valid whole number of seats.';
			case 'daily_rate_idr':
				return /^\d+$/.test(value) &&
					Number.isSafeInteger(fields.daily_rate_idr) &&
					fields.daily_rate_idr > 0
					? undefined
					: 'Enter a positive whole amount in IDR.';
		}
	}

	function fieldError(field: Field) {
		return touched[field] ? validate(field) : undefined;
	}

	function releasePreview(photo?: Photo) {
		if (photo?.url && previewUrls.delete(photo.url)) URL.revokeObjectURL(photo.url);
	}

	async function uploadPhoto(input: HTMLInputElement, kind: PhotoKind) {
		const file = input.files?.[0];
		input.value = '';
		if (!file || busy) return;
		if (file.size > 5 * 1024 * 1024) {
			uploadErrors[kind] = 'Choose an image no larger than 5 MiB.';
			return;
		}
		if (
			!file.size ||
			(file.type && !['image/jpeg', 'image/png', 'image/webp'].includes(file.type))
		) {
			uploadErrors[kind] = 'Choose a JPEG, PNG, or WebP image.';
			return;
		}

		uploadErrors[kind] = undefined;
		const controller = new AbortController();
		upload = controller;
		uploading = kind;
		try {
			const key = await uploadCatalogImage(file, controller.signal);
			if (controller.signal.aborted) return;
			const url = URL.createObjectURL(file);
			previewUrls.add(url);
			const brand = brandOptions.find((option) => option.value === fields.brand_id)?.label;
			const vehicle = [brand, fields.model].filter(Boolean).join(' ');
			const fallbackLabel =
				kind === 'cover' ? 'Cover photo' : `Gallery photo ${gallery.length + 1}`;
			const photo: Photo = {
				id: key,
				object_key: key,
				url,
				...suggestCarPhotoMetadata(file.name, vehicle, fallbackLabel)
			};
			if (kind === 'cover') {
				releasePreview(cover);
				cover = photo;
			} else {
				gallery.push(photo);
			}
		} catch (error) {
			if (!controller.signal.aborted) {
				uploadErrors[kind] =
					getApiError(error)?.message ?? 'Upload failed. Choose the image again to retry.';
			}
		} finally {
			upload = undefined;
			uploading = undefined;
		}
	}

	async function removeGalleryPhoto(photo: Photo) {
		gallery = gallery.filter((item) => item.id !== photo.id);
		releasePreview(photo);
		await tick();
		document.getElementById(`${ids}-gallery-picker`)?.focus();
	}

	async function goToStep(next: number) {
		if (busy || step === next) return;
		step = next;
		await tick();
		stepHeading.focus();
	}

	async function validateStep(index: number) {
		const invalid = steps[index].fields.filter(validate);
		if (!invalid.length) return true;
		for (const field of invalid) touched[field] = true;
		await goToStep(index);
		await tick();
		document.getElementById(`${ids}-${invalid[0]}`)?.focus();
		return false;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (blocked) return;
		if (step < imageStep) {
			const index = step;
			if ((await validateStep(index)) && step === index) await goToStep(index + 1);
			return;
		}

		for (let index = 0; index < imageStep; index += 1) {
			if (!(await validateStep(index))) return;
		}

		imageSubmitted = true;
		await tick();
		if (blocked || step !== imageStep) return;
		const invalidPhoto = form.querySelector<HTMLElement>(
			'[data-photos] :is([aria-invalid="true"], [data-missing])'
		);
		if (invalidPhoto) {
			invalidPhoto.focus();
			return;
		}

		let result;
		try {
			result = await save.mutateAsync();
		} catch (error) {
			toast({
				kind: 'error',
				message: getApiError(error)?.message ?? 'Could not save the car. Please try again.'
			});
			return;
		}
		queryClient.setQueryData(carDetailQuery(result.data.id).queryKey, result.data);
		void queryClient.invalidateQueries({ queryKey: [carsEndpoint] });
		toast({ kind: 'success', message: original ? 'Car updated.' : 'Car created.' });
		await goto(resolve('/admin/(staff)/cars/[id]', { id: result.data.id }));
	}
</script>

<form bind:this={form} class="mt-6 max-w-2xl" novalidate aria-busy={busy} onsubmit={submit}>
	<ol aria-label="Car form steps" class="grid grid-cols-4 gap-1 border-b border-line pb-3 sm:gap-2">
		{#each steps as item, index (item.label)}
			<li
				class="relative min-w-0 after:absolute after:top-4 after:left-[calc(50%+1.25rem)] after:w-[calc(100%-2rem)] after:border-t after:border-line last:after:content-none"
			>
				{#if index !== step && (original || index < step)}
					<button
						type="button"
						disabled={busy}
						onclick={() => goToStep(index)}
						class="inline-flex min-h-11 w-full flex-col items-center gap-2 rounded-lg text-center text-xs/4 font-medium text-ink-muted hover:text-primary disabled:cursor-not-allowed sm:text-sm"
					>
						{@render stepMarker(index)}
						{item.label}
					</button>
				{:else}
					<span
						aria-current={index === step ? 'step' : undefined}
						class={[
							'flex flex-col items-center gap-2 text-center text-xs/4 sm:text-sm',
							index === step ? 'font-semibold text-ink' : 'text-ink-muted'
						]}
					>
						{@render stepMarker(index)}
						{item.label}
					</span>
				{/if}
			</li>
		{/each}
	</ol>

	<section aria-labelledby={`${ids}-step-heading`} class="mt-6 min-w-0">
		<h2
			id={`${ids}-step-heading`}
			bind:this={stepHeading}
			tabindex="-1"
			class="mb-4 scroll-mt-24 text-lg font-semibold"
		>
			<span class="sr-only">Step {step + 1} of {steps.length}. </span>{currentStep.label}
		</h2>
		{#if step === 0}
			<div class="grid gap-4 sm:grid-cols-2">
				<div class="sm:col-span-2">
					<SelectField
						id={`${ids}-brand_id`}
						name="brand_id"
						label="Brand"
						placeholder={brands.isPending ? 'Loading brands…' : 'Select a brand'}
						options={brandOptions}
						bind:value={values.brand_id}
						required
						disabled={save.isPending || !brandOptions.length}
						error={fieldError('brand_id')}
						onchange={() => (touched.brand_id = true)}
					/>
					{#if brands.isError}
						<p class="mt-2 text-sm text-negative" role="alert">Could not load brands.</p>
						<Button variant="ghost" onclick={() => brands.refetch()}>Try again</Button>
					{:else if brands.isSuccess && !brandOptions.length}
						<p class="mt-2 text-sm text-ink-muted">Set up an active brand before creating a car.</p>
						<Button variant="ghost" href={resolve('/admin/(staff)/brands')}>Go to Brands</Button>
					{/if}
				</div>
				{@render text('model', 'Model', true, 100, 'e.g. Fortuner')}
				{@render text('trim', 'Trim', false, 100, 'e.g. 2.8 VRZ AT')}
			</div>
			<div class="mt-4">
				<label for={`${ids}-description`} class="mb-2 block text-sm font-medium">
					Description <span aria-hidden="true" class="text-negative">*</span>
				</label>
				<textarea
					id={`${ids}-description`}
					name="description"
					bind:value={values.description}
					rows="4"
					maxlength="2000"
					required
					disabled={save.isPending}
					onblur={() => (touched.description = true)}
					aria-invalid={fieldError('description') ? 'true' : undefined}
					aria-describedby={fieldError('description') ? `${ids}-description-error` : undefined}
					class={[
						'w-full resize-y rounded-xl border bg-panel px-4 py-3 text-base focus-visible:outline-offset-2 disabled:bg-panel-muted',
						fieldError('description') ? 'border-negative' : 'border-line hover:border-ink-muted'
					]}></textarea>
				<div aria-live="polite">
					{#if fieldError('description')}
						<p id={`${ids}-description-error`} class="mt-1 text-sm text-negative">
							{fieldError('description')}
						</p>
					{/if}
				</div>
			</div>
		{:else if step === 1}
			<div class="grid gap-4 sm:grid-cols-2">
				{@render select('category', 'Category', carCategories)}
				{@render select('body_type', 'Body type', carBodyTypes)}
				<NumberField
					id={`${ids}-seats`}
					name="seats"
					label="Seats"
					bind:value={values.seats}
					maxDigits={10}
					placeholder="e.g. 7"
					required
					disabled={save.isPending}
					onblur={() => (touched.seats = true)}
					error={fieldError('seats')}
				/>
				{@render select('transmission', 'Transmission', carTransmissions)}
				{@render select('fuel_type', 'Fuel type', carFuelTypes)}
			</div>
		{:else if step === 2}
			<div class="grid gap-4 sm:grid-cols-2">
				<NumberField
					id={`${ids}-daily_rate_idr`}
					name="daily_rate_idr"
					label="Daily rate"
					bind:value={values.daily_rate_idr}
					maxDigits={16}
					placeholder="e.g. 850.000"
					required
					disabled={save.isPending}
					onblur={() => (touched.daily_rate_idr = true)}
					error={fieldError('daily_rate_idr')}
				>
					{#snippet prepend()}IDR{/snippet}
				</NumberField>
				{@render select(
					'status',
					'Status',
					carStatuses,
					'Inactive cars are hidden from the public catalog.'
				)}
			</div>
		{:else}
			<div data-photos>
				<p class="mb-5 text-sm/6 text-ink-muted">
					JPEG, PNG, or WebP. Up to 5 MiB per photo. Review the suggested label and alt text after
					uploading.
				</p>
				<div>
					<h3 class="text-sm font-semibold">Cover photo</h3>
					<p class="mt-1 mb-3 text-xs/5 text-ink-muted">
						The cover is used on catalog cards and is the first image on the car page.
					</p>
					<input
						bind:this={coverInput}
						type="file"
						accept="image/jpeg,image/png,image/webp"
						hidden
						onchange={(event) => uploadPhoto(event.currentTarget, 'cover')}
					/>
					{#if cover}
						{#key cover.id}
							<CarPhotoFields
								photo={cover}
								src={cover.url}
								title="Cover photo"
								onedit={(field, value) => {
									if (cover) cover[field] = value;
								}}
								submitted={imageSubmitted}
								disabled={busy}
							/>
						{/key}
					{/if}
					{@render uploadButton('cover', cover ? 'Replace cover photo' : 'Add cover photo')}
				</div>
				<div class="mt-6 border-t border-line pt-5">
					<h3 class="text-sm font-semibold">Gallery photos</h3>
					<p class="mt-1 mb-3 text-xs/5 text-ink-muted">
						Add at least one additional view for the car detail page.
					</p>
					<input
						bind:this={galleryInput}
						type="file"
						accept="image/jpeg,image/png,image/webp"
						hidden
						onchange={(event) => uploadPhoto(event.currentTarget, 'gallery')}
					/>
					{#if gallery.length}
						<ol class="divide-y divide-line">
							{#each gallery as photo, index (photo.id)}
								<li class="py-4 first:pt-0">
									<p class="mb-2 text-xs font-medium text-ink-muted">Gallery photo {index + 1}</p>
									<CarPhotoFields
										{photo}
										src={photo.url}
										title={`Gallery photo ${index + 1}`}
										onedit={(field, value) => (photo[field] = value)}
										submitted={imageSubmitted}
										disabled={busy}
										onremove={() => removeGalleryPhoto(photo)}
									/>
								</li>
							{/each}
						</ol>
					{/if}
					{@render uploadButton('gallery', 'Add gallery photo')}
				</div>
			</div>
		{/if}
	</section>

	<AdminFormActions>
		<Button href={cancelHref} variant="ghost">Cancel</Button>
		{#if step > 0}
			<Button variant="secondary" disabled={busy} onclick={() => goToStep(step - 1)}>Back</Button>
		{/if}
		<Button type="submit" disabled={blocked}>
			{step < imageStep
				? 'Continue'
				: save.isPending
					? 'Saving…'
					: original
						? 'Save changes'
						: 'Create car'}
		</Button>
	</AdminFormActions>
</form>

{#snippet stepMarker(index: number)}
	<span
		aria-hidden="true"
		class={[
			'grid size-8 place-items-center rounded-full border text-sm font-medium tabular-nums',
			index === step
				? 'border-primary bg-primary text-on-primary'
				: 'border-line bg-panel text-ink-muted'
		]}
	>
		{#if !original && index < step}<Check size={15} strokeWidth={1.75} />{:else}{index + 1}{/if}
	</span>
{/snippet}

{#snippet text(
	name: 'model' | 'trim',
	label: string,
	required: boolean,
	maxlength: number,
	placeholder: string
)}
	<TextField
		id={`${ids}-${name}`}
		{name}
		{label}
		bind:value={values[name]}
		{required}
		{maxlength}
		{placeholder}
		disabled={save.isPending}
		onblur={() => (touched[name] = true)}
		error={fieldError(name)}
	/>
{/snippet}

{#snippet select(
	name: 'category' | 'body_type' | 'transmission' | 'fuel_type' | 'status',
	label: string,
	options: readonly { value: string; label: string }[],
	help?: string
)}
	<SelectField
		id={`${ids}-${name}`}
		{name}
		{label}
		{options}
		bind:value={values[name]}
		{help}
		required
		disabled={save.isPending}
		error={fieldError(name)}
		onchange={() => (touched[name] = true)}
	/>
{/snippet}

{#snippet uploadButton(kind: PhotoKind, label: string)}
	{@const missing =
		imageSubmitted && (kind === 'cover' ? needsCover && !cover : needsGallery && !gallery.length)}
	{@const error =
		uploadErrors[kind] ??
		(missing
			? `Add ${kind === 'cover' ? 'a cover photo' : 'at least one gallery photo'}.`
			: undefined)}
	{#if kind === 'cover'}
		<Button
			id={`${ids}-cover-picker`}
			variant="secondary"
			disabled={busy}
			onclick={() => coverInput?.click()}
			data-missing={missing || undefined}
			aria-describedby={error ? `${ids}-cover-error` : undefined}
			class={cover ? 'mt-3' : undefined}
		>
			<ImagePlus size={17} aria-hidden="true" />
			{uploading === kind ? 'Uploading…' : label}
		</Button>
	{:else}
		<Button
			id={`${ids}-gallery-picker`}
			variant="ghost"
			size="custom"
			disabled={busy}
			onclick={() => galleryInput?.click()}
			data-missing={missing || undefined}
			aria-describedby={error ? `${ids}-gallery-error` : undefined}
			class={[
				'min-h-16 w-full rounded-xl border border-dashed bg-panel-muted px-3',
				error ? 'border-negative' : 'border-line'
			]}
		>
			<ImagePlus size={17} aria-hidden="true" />
			{uploading === kind ? 'Uploading…' : label}
		</Button>
	{/if}
	<div aria-live="polite">
		{#if error}<p id={`${ids}-${kind}-error`} class="mt-2 text-sm text-negative">{error}</p>{/if}
	</div>
{/snippet}
