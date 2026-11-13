import { AxiosError, AxiosHeaders } from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { changePassword, login, revokeSession } from '#lib/server/api/identity.ts';
import { getCurrentUser } from '#lib/server/queries/identity.ts';
import { actions } from './login/+page.server';
import { load } from './(staff)/+layout.server';
import { actions as passwordActions } from './(staff)/profile/change-password/+page.server';
import { load as loadProfileFeedback } from './(staff)/profile/+page.server';
import { passwordUpdatedCookie } from './(staff)/profile/password-feedback';
import { POST } from './logout/+server';

vi.mock('#lib/server/api/identity.ts', () => ({
	changePassword: vi.fn(),
	login: vi.fn(),
	revokeSession: vi.fn()
}));
vi.mock('#lib/server/queries/identity.ts', () => ({ getCurrentUser: vi.fn() }));

const token = 'test-session-token';
const expiresAt = '2026-11-20T00:00:00Z';
const email = 'staff@example.test';
const password = '  test password  ';
const profile = { id: 'test-id', full_name: 'Test User', email, created_at: expiresAt };
const cookieName = 'sroom_staff_session';

function apiError(
	status: number,
	message: string,
	details?: Record<string, string>,
	code?: string
) {
	const config = { headers: new AxiosHeaders() };
	return new AxiosError(message, undefined, config, undefined, {
		status,
		statusText: '',
		data: { error: { message, details, code } },
		headers: new AxiosHeaders(),
		config
	});
}

function requestEvent(cookie?: string, credentials: Record<string, string> = { email, password }) {
	const url = new URL('https://sroom.example/admin/login');
	return {
		url,
		request: new Request(url, {
			method: 'POST',
			headers: { origin: url.origin },
			body: new URLSearchParams(credentials)
		}),
		cookies: { get: vi.fn(() => cookie), set: vi.fn(), delete: vi.fn() },
		setHeaders: vi.fn()
	};
}

const submit = (event: ReturnType<typeof requestEvent>) =>
	actions.default(event as unknown as Parameters<typeof actions.default>[0]);
const loadProfile = (event: ReturnType<typeof requestEvent>) =>
	load(event as unknown as Parameters<typeof load>[0]);
const signOut = (event: ReturnType<typeof requestEvent>) =>
	POST(event as unknown as Parameters<typeof POST>[0]);
const submitPassword = (event: ReturnType<typeof requestEvent>) =>
	passwordActions.default(event as unknown as Parameters<typeof passwordActions.default>[0]);

beforeEach(() => {
	vi.resetAllMocks();
	vi.mocked(login).mockResolvedValue({ token, expires_at: expiresAt });
	vi.mocked(revokeSession).mockResolvedValue(undefined);
	vi.mocked(changePassword).mockResolvedValue(undefined);
});

describe('staff change password', () => {
	const current_password = ' current password ';
	const new_password = ' new password ';
	const fields = { current_password, new_password, confirm_password: new_password };

	it('forwards only both unchanged passwords and preserves the current session', async () => {
		const event = requestEvent(token, { ...fields, user_id: 'other-user' });
		await expect(submitPassword(event)).rejects.toMatchObject({
			status: 303,
			location: '/admin/profile'
		});
		expect(changePassword).toHaveBeenCalledWith(token, { current_password, new_password });
		expect(event.cookies.delete).not.toHaveBeenCalled();
		expect(event.cookies.set).toHaveBeenCalledWith(
			passwordUpdatedCookie,
			'1',
			expect.objectContaining({ path: '/admin/profile', httpOnly: true, maxAge: 60 })
		);
	});

	it('consumes the success notice on the authenticated profile', async () => {
		const event = {
			...requestEvent('1'),
			parent: vi.fn().mockResolvedValue({ profile }),
			url: new URL('https://sroom.example/admin/profile')
		};
		const load = () =>
			loadProfileFeedback(event as unknown as Parameters<typeof loadProfileFeedback>[0]);
		await expect(load()).resolves.toEqual({
			toast: { kind: 'success', message: 'Password updated.' }
		});
		expect(event.cookies.delete).toHaveBeenCalledWith(passwordUpdatedCookie, {
			path: '/admin/profile'
		});
		event.cookies.get.mockReturnValue(undefined);
		await expect(load()).resolves.toEqual({});
	});

	it.each([
		['Confirm your new password.', { ...fields, confirm_password: '' }, 'confirm_password'],
		['Passwords do not match.', { ...fields, confirm_password: 'different' }, 'confirm_password'],
		[
			'Use at least 8 characters.',
			{ ...fields, new_password: '🔑'.repeat(4), confirm_password: '🔑'.repeat(4) },
			'new_password'
		]
	] as const)('reports %s without calling Identity', async (message, values, field) => {
		await expect(submitPassword(requestEvent(token, values))).resolves.toMatchObject({
			status: 400,
			data: { fields: { [field]: message } }
		});
		expect(changePassword).not.toHaveBeenCalled();
	});

	it('shows the incorrect-current-password response on its field without echoing inputs', async () => {
		vi.mocked(changePassword).mockRejectedValue(
			apiError(400, 'current password is incorrect', undefined, 'INVALID_CURRENT_PASSWORD')
		);
		const response = await submitPassword(requestEvent(token, fields));
		expect(response).toMatchObject({
			status: 400,
			data: { fields: { current_password: 'current password is incorrect' } }
		});
		expect(JSON.stringify(response)).not.toContain(current_password);
		expect(JSON.stringify(response)).not.toContain(new_password);
	});

	it('passes a multibyte password to Identity and displays its length error', async () => {
		const longPassword = 'é'.repeat(37);
		const message = 'password must not exceed 72 bytes';
		vi.mocked(changePassword).mockRejectedValue(
			apiError(400, 'validation failed', {
				new_password: message
			})
		);
		const response = await submitPassword(
			requestEvent(token, { ...fields, new_password: longPassword, confirm_password: longPassword })
		);
		expect(changePassword).toHaveBeenCalledWith(token, {
			current_password,
			new_password: longPassword
		});
		expect(response).toMatchObject({
			status: 400,
			data: { fields: { new_password: message }, toast: { kind: 'error', message } }
		});
	});

	it('preserves a forbidden response from Identity', async () => {
		vi.mocked(changePassword).mockRejectedValue(apiError(403, 'access denied'));
		await expect(submitPassword(requestEvent(token, fields))).resolves.toMatchObject({
			status: 403,
			data: { toast: { kind: 'error', message: 'access denied' } }
		});
	});

	it('rejects a cross-origin request before calling Identity', async () => {
		const event = requestEvent(token, fields);
		event.request.headers.set('origin', 'https://other.example');
		await expect(submitPassword(event)).resolves.toMatchObject({ status: 403 });
		expect(changePassword).not.toHaveBeenCalled();
	});

	it('redirects guests and clears expired sessions', async () => {
		const loginRedirect = { status: 303, location: '/admin/login' };
		await expect(submitPassword(requestEvent(undefined, fields))).rejects.toMatchObject(
			loginRedirect
		);
		expect(changePassword).not.toHaveBeenCalled();

		vi.mocked(changePassword).mockRejectedValue(apiError(401, 'authentication required'));
		const event = requestEvent(token, fields);
		await expect(submitPassword(event)).rejects.toMatchObject(loginRedirect);
		expect(event.cookies.delete).toHaveBeenCalledWith(cookieName, { path: '/admin' });
	});
});

describe('staff sign out', () => {
	it('rejects a cross-origin request before revoking the session', async () => {
		const event = requestEvent(token);
		event.request.headers.set('origin', 'https://other.example');
		await expect(signOut(event)).rejects.toMatchObject({ status: 403 });
		expect(revokeSession).not.toHaveBeenCalled();
	});

	it('revokes the session and clears its cookie before redirecting', async () => {
		const event = requestEvent(token);
		await expect(signOut(event)).rejects.toMatchObject({ status: 303, location: '/admin/login' });
		expect(revokeSession).toHaveBeenCalledWith(token);
		expect(event.cookies.delete).toHaveBeenCalledWith(cookieName, { path: '/admin' });
	});

	it('clears a session that has already expired or been revoked', async () => {
		vi.mocked(revokeSession).mockRejectedValue(apiError(401, 'authentication required'));
		const event = requestEvent(token);
		await expect(signOut(event)).rejects.toMatchObject({ status: 303, location: '/admin/login' });
		expect(event.cookies.delete).toHaveBeenCalledWith(cookieName, { path: '/admin' });
	});

	it('retains the cookie when Identity cannot revoke the session', async () => {
		vi.mocked(revokeSession).mockRejectedValue(apiError(503, 'unavailable'));
		const event = requestEvent(token);
		await expect(signOut(event)).rejects.toMatchObject({ status: 503 });
		expect(event.cookies.delete).not.toHaveBeenCalled();
	});
});

describe('staff login', () => {
	it.each(['OWNER', 'ADMIN'] as const)('starts a private session for %s', async (role) => {
		vi.mocked(getCurrentUser).mockResolvedValue({ ...profile, role });
		const event = requestEvent();
		await expect(submit(event)).rejects.toMatchObject({ status: 303, location: '/admin/profile' });
		expect(login).toHaveBeenCalledWith({ email, password });
		expect(getCurrentUser).toHaveBeenCalledWith(token);
		expect(event.cookies.set).toHaveBeenCalledWith(cookieName, token, {
			path: '/admin',
			httpOnly: true,
			sameSite: 'lax',
			secure: true,
			expires: new Date(expiresAt)
		});
	});

	it('revokes a customer session without setting a cookie', async () => {
		vi.mocked(getCurrentUser).mockResolvedValue({ ...profile, role: 'CUSTOMER' });
		const event = requestEvent();
		await expect(submit(event)).resolves.toMatchObject({
			status: 403,
			data: { email, message: 'Use a staff account to sign in here.' }
		});
		expect(revokeSession).toHaveBeenCalledWith(token);
		expect(event.cookies.set).not.toHaveBeenCalled();
	});

	it('validates a missing password before calling the gateway', async () => {
		const event = requestEvent(undefined, { email, password: '' });
		await expect(submit(event)).resolves.toMatchObject({
			status: 400,
			data: { email, fields: { password: 'Password is required.' } }
		});
		expect(login).not.toHaveBeenCalled();
	});

	it.each([
		[401, 'invalid email or password'],
		[429, 'too many requests'],
		[503, 'service unavailable']
	] as const)('uses the backend message for gateway status %s', async (status, message) => {
		vi.mocked(login).mockRejectedValue(apiError(status, message));
		const event = requestEvent();
		const result = await submit(event);
		expect(result).toMatchObject({ status, data: { email, message } });
		expect(JSON.stringify(result)).not.toContain(password);
		expect(JSON.stringify(result)).not.toContain(token);
		expect(event.cookies.set).not.toHaveBeenCalled();
	});

	it('uses backend validation details for the corresponding field', async () => {
		vi.mocked(login).mockRejectedValue(
			apiError(400, 'validation failed', { email: 'email must not exceed 255 characters' })
		);
		await expect(submit(requestEvent())).resolves.toMatchObject({
			status: 400,
			data: { fields: { email: 'email must not exceed 255 characters' } }
		});
	});

	it('shows a fallback when there is no response', async () => {
		vi.mocked(login).mockRejectedValue(new AxiosError('connection refused'));
		await expect(submit(requestEvent())).resolves.toMatchObject({
			status: 503,
			data: { message: 'Could not reach the sign-in service. Please try again.' }
		});
	});
});

describe('staff layout', () => {
	it('sends only the profile fields and disables caching', async () => {
		const user = { ...profile, role: 'ADMIN' as const, token };
		vi.mocked(getCurrentUser).mockResolvedValue(user);
		const event = requestEvent(token);
		await expect(loadProfile(event)).resolves.toEqual({
			profile: { full_name: profile.full_name, email, role: 'ADMIN' }
		});
		expect(event.setHeaders).toHaveBeenCalledWith({ 'cache-control': 'no-store' });
	});

	it('redirects a guest without making an Identity request', async () => {
		await expect(loadProfile(requestEvent())).rejects.toMatchObject({
			status: 303,
			location: '/admin/login'
		});
		expect(getCurrentUser).not.toHaveBeenCalled();
	});

	it('clears an expired session and redirects to login', async () => {
		vi.mocked(getCurrentUser).mockRejectedValue(apiError(401, 'authentication required'));
		const event = requestEvent(token);
		await expect(loadProfile(event)).rejects.toMatchObject({
			status: 303,
			location: '/admin/login'
		});
		expect(event.cookies.delete).toHaveBeenCalledWith(cookieName, { path: '/admin' });
	});

	it('returns 403 for a customer session', async () => {
		vi.mocked(getCurrentUser).mockResolvedValue({ ...profile, role: 'CUSTOMER' });
		await expect(loadProfile(requestEvent(token))).rejects.toMatchObject({ status: 403 });
	});

	it('preserves the cookie when the gateway is temporarily unavailable', async () => {
		vi.mocked(getCurrentUser).mockRejectedValue(apiError(504, 'service timed out'));
		const event = requestEvent(token);
		await expect(loadProfile(event)).rejects.toMatchObject({
			status: 504,
			body: { message: 'service timed out' }
		});
		expect(event.cookies.delete).not.toHaveBeenCalled();
	});
});
