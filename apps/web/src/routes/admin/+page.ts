import { resolve } from '$app/paths';
import { redirect } from '@sveltejs/kit';

export function load() {
	redirect(
		303,
		import.meta.env.DEV ? resolve('/admin/(staff)/overview') : resolve('/admin/(staff)/profile')
	);
}
