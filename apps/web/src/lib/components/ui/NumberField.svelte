<script lang="ts">
	import type { ComponentProps } from 'svelte';
	import TextField from './TextField.svelte';

	type Props = Omit<
		ComponentProps<typeof TextField>,
		'type' | 'inputmode' | 'maxlength' | 'leading' | 'onclear' | 'revealPassword'
	> & {
		name: string;
		maxDigits?: number;
	};

	let {
		value = $bindable(''),
		name,
		maxDigits,
		oninput,
		class: className,
		...props
	}: Props = $props();

	const formatter = new Intl.NumberFormat('id-ID');

	function digits(text: string) {
		return text.replace(/\D/g, '');
	}

	function trimLeadingZeros(text: string) {
		return text.replace(/^0+(?=\d)/, '');
	}

	function clean(text: string) {
		return trimLeadingZeros(digits(text)).slice(0, maxDigits);
	}

	function format(raw: string) {
		return raw ? formatter.format(BigInt(raw)) : '';
	}

	const raw = $derived(clean(value));
	const display = $derived(format(raw));

	function update(event: Parameters<NonNullable<Props['oninput']>>[0]) {
		const input = event.currentTarget;
		const text = input.value;
		const offset = input.selectionStart ?? text.length;
		const entered = digits(text);
		let next = clean(text);
		let before = digits(text.slice(0, offset)).length;

		// Deleting a grouping dot should delete the digit next to it.
		if (event instanceof InputEvent && next === raw && text.length === display.length - 1) {
			const remove =
				event.inputType === 'deleteContentBackward'
					? before - 1
					: event.inputType === 'deleteContentForward'
						? before
						: -1;
			if (remove >= 0) {
				next = clean(entered.slice(0, remove) + entered.slice(remove + 1));
				before = remove;
			}
		}

		const formatted = format(next);
		const leadingZeros = entered.length - trimLeadingZeros(entered).length;
		const digitsBefore = Math.min(next.length, Math.max(0, before - leadingZeros));
		let caret = 0;
		for (let count = 0; caret < formatted.length && count < digitsBefore; caret += 1) {
			if (formatted[caret] !== '.') count += 1;
		}
		value = next;
		input.value = formatted;
		input.setSelectionRange(caret, caret);
		oninput?.(event);
	}
</script>

<TextField
	{...props}
	inputmode="numeric"
	value={display}
	class={['tabular-nums', className]}
	oninput={update}
/>
<input type="hidden" {name} value={raw} disabled={props.disabled} form={props.form} />
