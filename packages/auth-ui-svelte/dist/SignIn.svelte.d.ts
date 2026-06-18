import type { AuthClient } from '@bigbase/auth';
interface Props {
    authClient: AuthClient;
    redirectTo?: string;
    mode?: 'email' | 'otp' | 'magic-link';
    showSocial?: boolean;
}
declare const SignIn: import("svelte").Component<Props, {}, "">;
type SignIn = ReturnType<typeof SignIn>;
export default SignIn;
