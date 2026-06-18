import type { AuthClient, AuthUser, AuthSession } from '@bigbase/auth';
import { createAuthClient } from '@bigbase/auth';
import { useState, useEffect, useCallback } from 'react';

export function useSession(client: AuthClient) {
  const [data, setData] = useState<AuthSession | null>(null);
  const [isPending, setIsPending] = useState(true);
  const [error, setError] = useState<Error | null>(null);

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

export function useAuth(options: { baseURL: string }) {
  const [client] = useState(() => createAuthClient(options));
  const session = useSession(client);
  const { data } = session;
  const user = data?.user ?? null;

  const signIn = useCallback((email: string, password: string) => client.signIn.email({ email, password }), [client]);
  const signUp = useCallback((email: string, password: string) => client.signUp.email({ email, password }), [client]);
  const signOut = useCallback(() => client.signOut(), [client]);

  return { client, user, session: data, isPending: session.isPending, error: session.error, signIn, signUp, signOut };
}

export { createAuthClient };
export type { AuthClient, AuthUser, AuthSession };
