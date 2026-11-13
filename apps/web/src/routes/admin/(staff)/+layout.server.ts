import { resolve } from '$app/paths';
import { error, redirect } from '@sveltejs/kit';
import { isAxiosError } from 'axios';
import { getCurrentUser } from '#lib/server/queries/identity.ts';
import { isStaff, staffCookiePath, staffSessionCookie } from '#lib/server/staff-session.ts';
import type { ApiError } from '#lib/types/identity.ts';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies, setHeaders }) => {
	setHeaders({ 'cache-control': 'no-store' });
	const token = cookies.get(staffSessionCookie);
	if (!token) redirect(303, resolve('/admin/login'));

	let user;
	try {
		user = await getCurrentUser(token);
	} catch (cause) {
		if (!isAxiosError<ApiError>(cause)) throw cause;
		if (cause.response?.status === 401) {
			cookies.delete(staffSessionCookie, { path: staffCookiePath });
			redirect(303, resolve('/admin/login'));
		}
		error(
			cause.response?.status ?? 503,
			cause.response?.data?.error?.message ??
				'Could not reach the profile service. Please try again.'
		);
	}

	if (!isStaff(user)) error(403, 'This page is for staff accounts.');
	return {
		profile: {
			full_name: user.full_name,
			email: user.email,
			role: user.role
		}
	};
};
