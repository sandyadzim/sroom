import { createContext } from 'svelte';

export const [useRegistrationEmail, setRegistrationEmail] = createContext<{ email: string }>();
