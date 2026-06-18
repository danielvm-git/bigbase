<script lang="ts">
  import type { AuthClient } from '@bigbase/auth';

  interface Props {
    authClient: AuthClient;
    redirectTo?: string;
  }

  let { authClient, redirectTo = '/' }: Props = $props();

  let name = $state('');
  let email = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleSignUp() {
    error = '';
    if (password.length < 6) { error = 'Password must be at least 6 characters'; return; }
    loading = true;
    try {
      await authClient.signUp.email({ email, password, name: name || undefined });
      window.location.href = redirectTo;
    } catch (e: any) {
      error = e.message || 'Sign up failed';
    } finally {
      loading = false;
    }
  }
</script>

<div class="bb-auth-signup">
  <h2>Create account</h2>
  <form onsubmit={(e) => { e.preventDefault(); handleSignUp(); }}>
    <input type="text" bind:value={name} placeholder="Name (optional)" disabled={loading} />
    <input type="email" bind:value={email} placeholder="Email" required disabled={loading} />
    <input type="password" bind:value={password} placeholder="Password (min 6 chars)" required minlength="6" disabled={loading} />
    <button type="submit" disabled={loading}>
      {loading ? 'Creating...' : 'Sign up'}
    </button>
  </form>
  {#if error}<p class="bb-auth-error">{error}</p>{/if}
</div>

<style>
  .bb-auth-signup { max-width: 360px; margin: 0 auto; }
  h2 { text-align: center; font-size: 20px; margin-bottom: 16px; }
  form { display: flex; flex-direction: column; gap: 10px; }
  input { padding: 10px; border: 1px solid #ccc; border-radius: 8px; font-size: 14px; }
  button { padding: 10px; background: #1a1a2e; color: white; border: none; border-radius: 8px; cursor: pointer; font-size: 14px; }
  button:disabled { opacity: 0.6; cursor: not-allowed; }
  .bb-auth-error { color: #e53e3e; font-size: 13px; text-align: center; }
</style>
