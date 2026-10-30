import { z } from 'zod';

export const loginSchema = z.object({
	email: z
		.string()
		.trim()
		.min(1, 'Email is required.')
		.pipe(z.email('Enter a valid email address.')),
	password: z.string().min(1, 'Password is required.')
});

export const registerSchema = loginSchema
	.extend({
		fullName: z.string().optional(),
		confirmPassword: z.string().min(1, 'Please confirm your password.')
	})
	.refine((values) => !values.confirmPassword || values.password === values.confirmPassword, {
		path: ['confirmPassword'],
		message: 'Passwords do not match.'
	});

export type AuthMode = 'login' | 'register';
export type AuthValues = z.input<typeof registerSchema>;
export type AuthField = 'email' | 'password' | 'confirmPassword';
export type AuthErrors = Partial<Record<AuthField, string>>;

export function validateAuthForm(mode: AuthMode, values: AuthValues): AuthErrors {
	const result = (mode === 'login' ? loginSchema : registerSchema).safeParse(values);
	const errors: AuthErrors = {};
	if (!result.success) {
		for (const issue of result.error.issues) {
			const field = issue.path[0] as AuthField;
			errors[field] ??= issue.message;
		}
	}
	return errors;
}

export function validateAuthField(
	field: AuthField,
	values: AuthValues,
	event: 'input' | 'blur'
): string | undefined {
	// Required errors wait for blur; format and matching errors update while typing.
	const empty = field === 'email' ? !values[field].trim() : !values[field];
	if (event === 'input' && (field === 'password' || empty)) return;
	return validateAuthForm('register', values)[field];
}
