import type { AuthClient } from '@bigbase/auth';
interface Props {
    authClient: AuthClient;
    path?: 'sign-in' | 'sign-up';
    redirectTo?: string;
}
declare const AuthView: import("svelte").Component<Props, {}, "">;
type AuthView = ReturnType<typeof AuthView>;
export default AuthView;
