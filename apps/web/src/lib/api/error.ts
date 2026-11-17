import { isAxiosError } from 'axios';
import type { ApiError } from '#lib/types/identity.ts';

export function getApiError(cause: unknown) {
	return isAxiosError<ApiError>(cause) ? cause.response?.data?.error : undefined;
}
