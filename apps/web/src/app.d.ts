import type { ToastFeedback } from '#lib/ui/toast.ts';

// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		interface PageData {
			toast?: ToastFeedback;
		}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
