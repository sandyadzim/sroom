import type { Handle } from '@sveltejs/kit/hooks';

const noindex = 'noindex, nofollow, noarchive, nosnippet, noimageindex';

export const handle: Handle = async ({ event, resolve }) => {
	const response = await resolve(event);
	response.headers.set('X-Robots-Tag', noindex);
	return response;
};
