import { resolve } from '$app/paths';
import { error, redirect } from '@sveltejs/kit';
import { isAxiosError } from 'axios';
import { revokeSession } from '#lib/server/api/identity.ts';
import { staffCookiePath, staffSessionCookie } from '#lib/server/staff-session.ts';
import type { RequestHandler } from './$types';

export const POST: RequestHandler = async ({ request, url, cookies, setHeaders }) => {
	setHeaders({ 'cache-control': 'no-store' });
	if (request.headers.get('origin') !== url.origin) error(403, 'Invalid sign-out request.');

	const token = cookies.get(staffSessionCookie);
	if (token) {
		try {
			await revokeSession(token);
		} catch (cause) {
			if (!isAxiosError(cause)) throw cause;
			if (cause.response?.status !== 401) error(503, 'Could not sign out. Please try again.');
		}
		cookies.delete(staffSessionCookie, { path: staffCookiePath });
	}

	redirect(303, resolve('/admin/login'));
};
