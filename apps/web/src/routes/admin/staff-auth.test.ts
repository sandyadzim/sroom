import { AxiosError, AxiosHeaders } from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { login, revokeSession } from '#lib/server/api/identity.ts';
import { getCurrentUser } from '#lib/server/queries/identity.ts';
import { actions } from './login/+page.server';
import { load } from './profile/+page.server';

vi.mock('#lib/server/api/identity.ts', () => ({ login: vi.fn(), revokeSession: vi.fn() }));
vi.mock('#lib/server/queries/identity.ts', () => ({ getCurrentUser: vi.fn() }));

const token = 'test-session-token';
const expiresAt = '2026-11-20T00:00:00Z';
const email = 'staff@example.test';
const password = '  test password  ';
const profile = { id: 'test-id', full_name: 'Test User', email, created_at: expiresAt };
const cookieName = 'sroom_staff_session';

function apiError(status: number, message: string, details?: Record<string, string>) {
	const config = { headers: new AxiosHeaders() };
	return new AxiosError(message, undefined, config, undefined, {
		status,
		statusText: '',
		data: { error: { message, details } },
		headers: new AxiosHeaders(),
		config
	});
}

function requestEvent(cookie?: string, credentials = { email, password }) {
	const url = new URL('https://sroom.example/admin/login');
	return {
		url,
		request: new Request(url, { method: 'POST', body: new URLSearchParams(credentials) }),
		cookies: { get: vi.fn(() => cookie), set: vi.fn(), delete: vi.fn() },
		setHeaders: vi.fn()
	};
}

const submit = (event: ReturnType<typeof requestEvent>) =>
	actions.default(event as unknown as Parameters<typeof actions.default>[0]);
const loadProfile = (event: ReturnType<typeof requestEvent>) =>
	load(event as unknown as Parameters<typeof load>[0]);

beforeEach(() => {
	vi.resetAllMocks();
	vi.mocked(login).mockResolvedValue({ token, expires_at: expiresAt });
	vi.mocked(revokeSession).mockResolvedValue(undefined);
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

describe('staff profile', () => {
	it('sends only the profile fields and disables caching', async () => {
		const user = { ...profile, role: 'ADMIN' as const, token };
		vi.mocked(getCurrentUser).mockResolvedValue(user);
		const event = requestEvent(token);
		await expect(loadProfile(event)).resolves.toEqual({ profile: { ...profile, role: 'ADMIN' } });
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
