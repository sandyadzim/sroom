import { passwordUpdatedCookie } from './password-feedback';
import type { PageServerLoad } from './$types';

export const load = (async ({ cookies, parent, url }) => {
	await parent();
	const passwordUpdated = cookies.get(passwordUpdatedCookie) === '1';
	if (passwordUpdated) cookies.delete(passwordUpdatedCookie, { path: url.pathname });
	return passwordUpdated ? { toast: { kind: 'success', message: 'Password updated.' } } : {};
}) satisfies PageServerLoad;
