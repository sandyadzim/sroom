import { resolve } from '$app/paths';
import { fail, redirect } from '@sveltejs/kit';
import { isAxiosError } from 'axios';
import { z } from 'zod';
import { changePassword } from '#lib/server/api/identity.ts';
import { staffCookiePath, staffSessionCookie } from '#lib/server/staff-session.ts';
import type { ApiError } from '#lib/types/identity.ts';
import type { ToastFeedback } from '#lib/ui/toast.ts';
import { changePasswordSchema, readPasswordForm } from '#lib/validation/change-password.ts';
import { passwordUpdatedCookie } from '../password-feedback';
import type { Actions } from './$types';

function errorToast(message: string): ToastFeedback {
	return { kind: 'error', message };
}

export const actions = {
	default: async ({ request, url, cookies }) => {
		if (request.headers.get('origin') !== url.origin) {
			return fail(403, {
				toast: errorToast('Invalid password change request.'),
				fields: undefined
			});
		}

		const token = cookies.get(staffSessionCookie);
		if (!token) redirect(303, resolve('/admin/login'));

		const result = changePasswordSchema.safeParse(readPasswordForm(await request.formData()));
		if (!result.success) {
			const fields = z.flattenError(result.error).fieldErrors;
			return fail(400, {
				toast: {
					kind: 'warning',
					message: 'Check the highlighted fields and try again.'
				} satisfies ToastFeedback,
				fields: {
					current_password: fields.current_password?.[0],
					new_password: fields.new_password?.[0],
					confirm_password: fields.confirm_password?.[0]
				}
			});
		}

		const { current_password, new_password } = result.data;
		try {
			await changePassword(token, { current_password, new_password });
		} catch (cause) {
			if (!isAxiosError<ApiError>(cause)) throw cause;
			const status = cause.response?.status ?? 503;
			if (status === 401) {
				cookies.delete(staffSessionCookie, { path: staffCookiePath });
				redirect(303, resolve('/admin/login'));
			}

			const apiError = cause.response?.data?.error;
			const currentError =
				apiError?.code === 'INVALID_CURRENT_PASSWORD'
					? apiError.message
					: apiError?.details?.current_password;
			const newError = apiError?.details?.new_password;
			return fail(status, {
				fields: {
					current_password: currentError,
					new_password: newError,
					confirm_password: undefined
				},
				toast: errorToast(
					currentError ??
						newError ??
						apiError?.message ??
						'Could not change your password. Please try again.'
				)
			});
		}

		const profilePath = resolve('admin/profile');
		cookies.set(passwordUpdatedCookie, '1', {
			path: new URL(profilePath, url).pathname,
			httpOnly: true,
			sameSite: 'lax',
			secure: url.protocol === 'https:',
			maxAge: 60
		});
		redirect(303, profilePath);
	}
} satisfies Actions;
