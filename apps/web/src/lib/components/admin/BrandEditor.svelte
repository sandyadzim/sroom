<script lang="ts">
	import { Eye, LoaderCircle, Plus, Trash2 } from '@lucide/svelte';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { onDestroy } from 'svelte';
	import {
		brandsEndpoint,
		createBrand,
		updateBrand,
		uploadCatalogImage
	} from '#lib/api/catalog/brand.ts';
	import { getApiError } from '#lib/api/staff.ts';
	import Button from '#lib/components/ui/Button.svelte';
	import Dialog from '#lib/components/ui/Dialog.svelte';
	import ImagePreviewDialog from '#lib/components/ui/ImagePreviewDialog.svelte';
	import ImageWithPlaceholder from '#lib/components/ui/ImageWithPlaceholder.svelte';
	import TextField from '#lib/components/ui/TextField.svelte';
	import type { Brand, BrandStatus, UpdateBrandInput } from '#lib/types/catalog.ts';
	import { useToast } from '#lib/ui/toast.ts';

	let { brand }: { brand?: Brand } = $props();
	const editorId = $props.id();
	const imageId = `${editorId}-image`;
	const queryClient = useQueryClient();
	const showToast = useToast();
	const imageTypes = ['image/jpeg', 'image/png', 'image/webp'];

	let open = $state(false);
	let original = $state.raw<Brand>();
	let name = $state('');
	let active = $state(true);
	let uploadedImage = $state.raw<{ file: File; key: string }>();
	let pendingUpload = $state.raw<AbortController>();
	let fileInput = $state<HTMLInputElement>();
	let previewUrl = $state<string>();
	let removeImage = $state(false);
	let nameError = $state<string>();
	let imageError = $state<string>();
	const status: BrandStatus = $derived(active ? 'active' : 'inactive');
	const imageName = $derived(
		uploadedImage?.file.name ??
			(!removeImage && original?.image_object_key
				? original.image_filename || original.image_object_key.split('/').at(-1)
				: undefined)
	);
	const imageUrl = $derived(
		uploadedImage ? previewUrl : removeImage ? undefined : original?.image_url
	);
	const hasChanges = $derived(
		!original ||
			name.trim() !== original.name ||
			status !== original.status ||
			!!uploadedImage ||
			removeImage
	);

	onDestroy(() => pendingUpload?.abort());

	$effect(() => {
		if (!open || !uploadedImage) return;
		const url = URL.createObjectURL(uploadedImage.file);
		previewUrl = url;
		return () => {
			URL.revokeObjectURL(url);
			previewUrl = undefined;
		};
	});

	const saveBrand = createMutation(() => ({
		mutationFn: () => {
			const trimmedName = name.trim();
			const previous = original;

			if (!previous) {
				return createBrand({
					name: trimmedName,
					image_object_key: uploadedImage?.key,
					image_filename: uploadedImage?.file.name
				});
			}

			const changes: UpdateBrandInput = {};
			if (trimmedName !== previous.name) changes.name = trimmedName;
			if (status !== previous.status) changes.status = status;
			if (uploadedImage) {
				changes.image_object_key = uploadedImage.key;
				changes.image_filename = uploadedImage.file.name;
			} else if (removeImage) {
				changes.image_object_key = null;
			}
			return updateBrand(previous.id, changes);
		}
	}));
	const busy = $derived(saveBrand.isPending || !!pendingUpload);

	function setOpen(next: boolean) {
		if (next) {
			original = brand;
			name = brand?.name ?? '';
			active = brand?.status !== 'inactive';
			removeImage = false;
			nameError = undefined;
		} else {
			pendingUpload?.abort();
			pendingUpload = undefined;
			uploadedImage = undefined;
			imageError = undefined;
		}
		open = next;
	}

	function validateName(value: string) {
		const trimmed = value.trim();
		if (!trimmed) return 'Enter a brand name.';
		if (Array.from(trimmed).length > 100) return 'Use at most 100 characters.';
	}

	async function selectFile(input: HTMLInputElement) {
		const selected = input.files?.[0];
		input.value = '';
		if (!selected || busy) return;

		imageError = undefined;
		if (selected.size > 5 * 1024 * 1024) {
			imageError = 'Choose an image no larger than 5 MiB.';
		} else if (!selected.size || (selected.type && !imageTypes.includes(selected.type))) {
			imageError = 'Choose a JPEG, PNG, or WebP image.';
		} else if (Array.from(selected.name).length > 255) {
			imageError = 'Use a file name with at most 255 characters.';
		}
		if (imageError) return;

		const upload = new AbortController();
		pendingUpload = upload;
		try {
			const key = await uploadCatalogImage(selected, upload.signal);
			if (!upload.signal.aborted) uploadedImage = { file: selected, key };
		} catch (cause) {
			if (!upload.signal.aborted) {
				imageError = getApiError(cause)?.message ?? 'Could not upload the image. Please try again.';
			}
		} finally {
			if (pendingUpload === upload) pendingUpload = undefined;
		}
	}

	function removeSelectedImage() {
		imageError = undefined;
		if (uploadedImage) uploadedImage = undefined;
		else removeImage = true;
		document.getElementById(`${imageId}-choose`)?.focus();
	}

	async function submit(event: SubmitEvent & { currentTarget: HTMLFormElement }) {
		event.preventDefault();
		if (busy || !hasChanges) return;

		const nameInput = event.currentTarget.querySelector<HTMLInputElement>('input[name="name"]');
		nameError = validateName(name);
		if (nameError || imageError) {
			showToast({ kind: 'warning', message: 'Check the highlighted fields and try again.' });
			(nameError ? nameInput : document.getElementById(`${imageId}-choose`))?.focus();
			return;
		}

		try {
			const saved = await saveBrand.mutateAsync();
			setOpen(false);
			showToast({ kind: 'success', message: saved.message });
			void queryClient.invalidateQueries({ queryKey: [brandsEndpoint] });
		} catch (cause) {
			const error = getApiError(cause);
			if (open && error?.code === 'BRAND_NAME_TAKEN') {
				nameError = error.message;
				nameInput?.focus();
			}
			showToast({
				kind: 'error',
				message: error?.message ?? 'Could not save this brand. Please try again.'
			});
		}
	}
</script>

<Dialog
	title={brand ? 'Edit brand' : 'Create brand'}
	description={brand ? 'Update the brand name, status, or image.' : 'Add a brand to the catalog.'}
	size="compact"
	bind:open={() => open, setOpen}
>
	{#snippet trigger({ props })}
		{#if brand}
			<Button
				{...props}
				variant="ghost"
				size="custom"
				class="rounded-lg px-3"
				aria-label={`Edit ${brand.name}`}
				disabled={saveBrand.isPending}
			>
				Edit
			</Button>
		{:else}
			<Button {...props} disabled={saveBrand.isPending}>
				<Plus size={18} strokeWidth={1.5} aria-hidden="true" />
				Create brand
			</Button>
		{/if}
	{/snippet}

	<form method="post" novalidate aria-busy={saveBrand.isPending} onsubmit={submit}>
		<div class="space-y-5">
			<TextField
				label="Brand name"
				name="name"
				value={name}
				error={nameError}
				disabled={saveBrand.isPending}
				oninput={(event) => {
					name = event.currentTarget.value;
					nameError = undefined;
				}}
				onblur={() => (nameError ??= validateName(name))}
				required
			/>

			{#if original}
				<fieldset disabled={saveBrand.isPending}>
					<legend class="mb-2 text-sm font-medium">Status</legend>
					<label class="flex min-h-11 w-fit cursor-pointer items-center gap-3 text-sm">
						<input type="checkbox" bind:checked={active} class="size-5 accent-primary" />
						Active
					</label>
				</fieldset>
			{/if}

			<div>
				<label for={imageId} class="mb-2 block text-sm font-medium">Brand image (optional)</label>
				<input
					bind:this={fileInput}
					id={imageId}
					name="image"
					type="file"
					tabindex="-1"
					class="sr-only"
					accept={imageTypes.join(',')}
					disabled={busy}
					onchange={(event) => selectFile(event.currentTarget)}
				/>
				<div class="flex flex-wrap gap-3">
					{#if imageName}
						<div
							class="group relative flex size-32 flex-col overflow-hidden rounded-xl border border-line bg-panel"
						>
							<ImageWithPlaceholder src={imageUrl} alt="" class="min-h-0 flex-1" />
							<span class="truncate px-2 py-1.5 text-sm" title={imageName}>{imageName}</span>
							<div
								class="absolute top-1.5 right-1.5 flex gap-0.5 rounded-lg bg-panel p-0.5 motion-safe:transition-opacity [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-focus-within:opacity-100 [@media(hover:hover)]:group-hover:opacity-100"
							>
								<ImagePreviewDialog title="Brand Image" filename={imageName} src={imageUrl}>
									{#snippet trigger({ props })}
										<Button
											{...props}
											variant="ghost"
											size="custom"
											class="size-8 min-h-0! rounded-md"
											aria-label="View image"
											title="View image"
											disabled={saveBrand.isPending}
										>
											<Eye size={16} strokeWidth={1.5} aria-hidden="true" />
										</Button>
									{/snippet}
								</ImagePreviewDialog>
								<Button
									variant="ghost"
									size="custom"
									class="size-8 min-h-0! rounded-md"
									aria-label="Remove image"
									title="Remove image"
									disabled={busy}
									onclick={removeSelectedImage}
								>
									<Trash2 size={16} strokeWidth={1.5} aria-hidden="true" />
								</Button>
							</div>
						</div>
					{/if}
					<Button
						id={`${imageId}-choose`}
						variant="ghost"
						size="custom"
						disabled={busy}
						aria-invalid={imageError ? 'true' : undefined}
						aria-describedby={imageError ? `${imageId}-help ${imageId}-error` : `${imageId}-help`}
						class={[
							'size-32 flex-col gap-2! rounded-xl border border-dashed bg-page text-ink-muted!',
							imageError ? 'border-negative' : 'border-line'
						]}
						onclick={() => fileInput?.click()}
					>
						{#if pendingUpload}
							<LoaderCircle
								size={24}
								strokeWidth={1.5}
								class="motion-safe:animate-spin"
								aria-hidden="true"
							/>
							Uploading…
						{:else}
							<Plus size={24} strokeWidth={1.5} aria-hidden="true" />
							{imageName ? 'Replace image' : 'Add image'}
						{/if}
					</Button>
				</div>
				<p id={`${imageId}-help`} class="mt-2 text-sm text-ink-muted">
					JPEG, PNG, or WebP up to 5 MiB.
				</p>
				<div aria-live="polite">
					<span class="sr-only"
						>{pendingUpload ? 'Uploading image…' : uploadedImage ? 'Image uploaded.' : ''}</span
					>
					{#if imageError}<p id={`${imageId}-error`} class="mt-2 text-sm text-negative">
							{imageError}
						</p>
						<Button
							variant="ghost"
							size="custom"
							class="rounded-lg px-2"
							onclick={() => (imageError = undefined)}>Dismiss</Button
						>
					{/if}
				</div>
				{#if removeImage && !uploadedImage}
					<div class="mt-1 flex flex-wrap items-center gap-x-2 text-sm">
						<span class="text-ink-muted">Image will be removed on save.</span>
						<Button
							variant="ghost"
							size="custom"
							class="rounded-lg px-2"
							disabled={busy}
							onclick={() => (removeImage = false)}>Undo removal</Button
						>
					</div>
				{/if}
			</div>
		</div>

		<div class="mt-6 flex flex-wrap justify-end gap-2 border-t border-line pt-4">
			<Button variant="secondary" disabled={saveBrand.isPending} onclick={() => setOpen(false)}>
				Cancel
			</Button>
			<Button type="submit" disabled={busy || !hasChanges}>
				{saveBrand.isPending ? 'Saving…' : brand ? 'Save changes' : 'Create brand'}
			</Button>
		</div>
	</form>
</Dialog>
