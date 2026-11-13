export type LoginCredentials = {
	email: string;
	password: string;
};

export type LoginSession = {
	token: string;
	expires_at: string;
};

export type UserProfile = {
	id: string;
	full_name: string;
	email: string;
	role: 'CUSTOMER' | 'ADMIN' | 'OWNER';
	created_at: string;
};

export type ApiData<T> = { data: T };

export type ApiError = {
	error: {
		message: string;
		details?: Record<string, string>;
	};
};
