import { resolve } from '$app/paths';
import { fail, redirect } from '@sveltejs/kit';
import { isAxiosError } from 'axios';
import { z } from 'zod';
import { login, revokeSession } from '#lib/server/api/identity.ts';
import { getCurrentUser } from '#lib/server/queries/identity.ts';
import { isStaff, staffCookiePath, staffSessionCookie } from '#lib/server/staff-session.ts';
import type { ApiError, LoginSession } from '#lib/types/identity.ts';
import { loginSchema } from '#lib/validation/auth.ts';
import type { Actions } from './$types';

export const actions: Actions = {
	default: async ({ request, cookies, url, setHeaders }) => {
		setHeaders({ 'cache-control': 'no-store' });
		const form = await request.formData();
		const email = String(form.get('email') ?? '').trim();
		const credentials = loginSchema.safeParse({ email, password: form.get('password') ?? '' });
		if (!credentials.success) {
			const fields = z.flattenError(credentials.error).fieldErrors;
			return fail(400, {
				email,
				fields: { email: fields.email?.[0], password: fields.password?.[0] }
			});
		}

		let session: LoginSession | undefined;
		try {
			session = await login(credentials.data);
			const user = await getCurrentUser(session.token);
			if (!isStaff(user)) {
				await revokeSession(session.token).catch(() => {});
				return fail(403, { email, message: 'Use a staff account to sign in here.' });
			}
		} catch (cause) {
			if (session) await revokeSession(session.token).catch(() => {});
			if (!isAxiosError<ApiError>(cause)) throw cause;
			const apiError = cause.response?.data?.error;
			return fail(cause.response?.status ?? 503, {
				email,
				fields: { email: apiError?.details?.email, password: apiError?.details?.password },
				message: apiError?.message ?? 'Could not reach the sign-in service. Please try again.'
			});
		}

		cookies.set(staffSessionCookie, session.token, {
			path: staffCookiePath,
			httpOnly: true,
			sameSite: 'lax',
			secure: url.protocol === 'https:',
			expires: new Date(session.expires_at)
		});
		redirect(303, resolve('/admin/profile'));
	}
};
