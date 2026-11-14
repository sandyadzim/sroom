import { z } from 'zod';

export const changePasswordSchema = z
	.object({
		current_password: z.string().min(1, 'Enter your current password.'),
		new_password: z
			.string()
			.refine((value) => Array.from(value).length >= 8, 'Use at least 8 characters.'),
		confirm_password: z.string().min(1, 'Confirm your new password.')
	})
	.refine((values) => !values.confirm_password || values.new_password === values.confirm_password, {
		path: ['confirm_password'],
		message: 'Passwords do not match.'
	});

export type PasswordField = keyof z.input<typeof changePasswordSchema>;
export type PasswordErrors = Partial<Record<PasswordField, string>>;

export function readPasswordForm(form: FormData) {
	return {
		current_password: String(form.get('current_password') ?? ''),
		new_password: String(form.get('new_password') ?? ''),
		confirm_password: String(form.get('confirm_password') ?? '')
	};
}

export function validatePasswordForm(values: ReturnType<typeof readPasswordForm>): PasswordErrors {
	const result = changePasswordSchema.safeParse(values);
	if (result.success) return {};
	const fields = z.flattenError(result.error).fieldErrors;
	return {
		current_password: fields.current_password?.[0],
		new_password: fields.new_password?.[0],
		confirm_password: fields.confirm_password?.[0]
	};
}

export function validatePasswordField(
	field: PasswordField,
	values: ReturnType<typeof readPasswordForm>,
	event: 'input' | 'blur'
) {
	if (event === 'input' && !values[field]) return;
	return validatePasswordForm(values)[field];
}
