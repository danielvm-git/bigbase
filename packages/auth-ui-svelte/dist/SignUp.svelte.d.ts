import type { AuthClient } from '@bigbase/auth';
interface Props {
    authClient: AuthClient;
    redirectTo?: string;
}
declare const SignUp: import("svelte").Component<Props, {}, "">;
type SignUp = ReturnType<typeof SignUp>;
export default SignUp;
