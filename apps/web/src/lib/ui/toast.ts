import { createContext } from 'svelte';

export type ToastFeedback = {
	kind: 'success' | 'error' | 'warning';
	message: string;
};

export type ToastItem = ToastFeedback & { id: number };

export const [useToast, setToastContext] = createContext<(feedback: ToastFeedback) => void>();
