import type { AuthClient } from '@bigbase/auth';
interface Props {
    authClient: AuthClient;
    loginUrl?: string;
}
declare const UserButton: import("svelte").Component<Props, {}, "">;
type UserButton = ReturnType<typeof UserButton>;
export default UserButton;
