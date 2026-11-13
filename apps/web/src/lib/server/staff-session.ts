import type { UserProfile } from '#lib/types/identity.ts';

export const staffSessionCookie = 'sroom_staff_session';
export const staffCookiePath = '/admin';

export function isStaff(user: UserProfile) {
	return user.role === 'ADMIN' || user.role === 'OWNER';
}
