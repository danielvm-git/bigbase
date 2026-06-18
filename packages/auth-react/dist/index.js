import { createAuthClient } from '@bigbase/auth';
import { useState, useEffect, useCallback } from 'react';
export function useSession(client) {
    const [data, setData] = useState(null);
    const [isPending, setIsPending] = useState(true);
    const [error, setError] = useState(null);
    useEffect(() => {
        client.getSession().then(setData).catch(setError).finally(() => setIsPending(false));
        const unsub = client.onAuthStateChange((s) => {
            setData(s);
            setIsPending(false);
        });
        return unsub;
    }, [client]);
    return { data, isPending, error };
}
export function useAuth(options) {
    const [client] = useState(() => createAuthClient(options));
    const session = useSession(client);
    const { data } = session;
    const user = data?.user ?? null;
    const signIn = useCallback((email, password) => client.signIn.email({ email, password }), [client]);
    const signUp = useCallback((email, password) => client.signUp.email({ email, password }), [client]);
    const signOut = useCallback(() => client.signOut(), [client]);
    return { client, user, session: data, isPending: session.isPending, error: session.error, signIn, signUp, signOut };
}
export { createAuthClient };
