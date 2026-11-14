import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	PUBLIC_GATEWAY_BASE_URL: {
		public: true,
		static: true,
		schema: (value) => value || ''
	}
});
