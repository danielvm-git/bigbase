<script lang="ts">
  import type { AuthClient } from '@bigbase/auth';

  interface Props {
    authClient: AuthClient;
    redirectTo?: string;
    mode?: 'email' | 'otp' | 'magic-link';
    showSocial?: boolean;
  }

  let { authClient, redirectTo = '/', mode = 'email', showSocial = true }: Props = $props();

  let email = $state('');
  let password = $state('');
  let otpCode = $state('');
  let otpSent = $state(false);
  let magicLinkSent = $state(false);
  let error = $state('');
  let loading = $state(false);

  async function handleEmailSignIn() {
    error = '';
    loading = true;
    try {
      await authClient.signIn.email({ email, password });
      window.location.href = redirectTo;
    } catch (e: any) {
      error = e.message || 'Sign in failed';
    } finally {
      loading = false;
    }
  }

  async function handleOTPSend() {
    error = '';
    loading = true;
    try {
      await authClient.signIn.otp.send({ email });
      otpSent = true;
    } catch (e: any) {
      error = e.message || 'Failed to send code';
    } finally {
      loading = false;
    }
  }

  async function handleOTPVerify() {
    error = '';
    loading = true;
    try {
      await authClient.signIn.otp.verify({ email, code: otpCode });
      window.location.href = redirectTo;
    } catch (e: any) {
      error = e.message || 'Invalid code';
    } finally {
      loading = false;
    }
  }

  async function handleMagicLinkSend() {
    error = '';
    loading = true;
    try {
      await authClient.signIn.magicLink.send({ email, redirectTo });
      magicLinkSent = true;
    } catch (e: any) {
      error = e.message || 'Failed to send magic link';
    } finally {
      loading = false;
    }
  }

  function handleSocial() {
    authClient.signIn.social({ provider: 'google', redirectTo });
  }
</script>

<div class="bb-auth-signin">
  {#if showSocial}
    <button class="bb-auth-social-btn" onclick={handleSocial} disabled={loading}>
      Sign in with Google
    </button>
    <div class="bb-auth-divider"><span>or</span></div>
  {/if}

  {#if mode === 'email'}
    <form onsubmit={(e) => { e.preventDefault(); handleEmailSignIn(); }}>
      <input type="email" bind:value={email} placeholder="Email" required disabled={loading} />
      <input type="password" bind:value={password} placeholder="Password" required disabled={loading} />
      <button type="submit" disabled={loading}>
        {loading ? 'Signing in...' : 'Sign in'}
      </button>
    </form>
  {:else if mode === 'otp'}
    {#if !otpSent}
      <form onsubmit={(e) => { e.preventDefault(); handleOTPSend(); }}>
        <input type="email" bind:value={email} placeholder="Email" required disabled={loading} />
        <button type="submit" disabled={loading}>
          {loading ? 'Sending...' : 'Send code'}
        </button>
      </form>
    {:else}
      <form onsubmit={(e) => { e.preventDefault(); handleOTPVerify(); }}>
        <p class="bb-auth-hint">Code sent to {email}</p>
        <input type="text" bind:value={otpCode} placeholder="6-digit code" maxlength="6" required disabled={loading} />
        <button type="submit" disabled={loading}>
          {loading ? 'Verifying...' : 'Verify'}
        </button>
      </form>
    {/if}
  {:else if mode === 'magic-link'}
    {#if !magicLinkSent}
      <form onsubmit={(e) => { e.preventDefault(); handleMagicLinkSend(); }}>
        <input type="email" bind:value={email} placeholder="Email" required disabled={loading} />
        <button type="submit" disabled={loading}>
          {loading ? 'Sending...' : 'Send magic link'}
        </button>
      </form>
    {:else}
      <p class="bb-auth-hint">Check your email for a sign-in link.</p>
    {/if}
  {/if}

  {#if error}
    <p class="bb-auth-error">{error}</p>
  {/if}
</div>

<style>
  .bb-auth-signin { max-width: 360px; margin: 0 auto; }
  .bb-auth-social-btn { width: 100%; padding: 10px; border: 1px solid #ccc; border-radius: 8px; background: white; cursor: pointer; font-size: 14px; }
  .bb-auth-divider { text-align: center; margin: 16px 0; color: #999; }
  .bb-auth-divider span { background: white; padding: 0 8px; }
  form { display: flex; flex-direction: column; gap: 10px; }
  input { padding: 10px; border: 1px solid #ccc; border-radius: 8px; font-size: 14px; }
  button { padding: 10px; background: #1a1a2e; color: white; border: none; border-radius: 8px; cursor: pointer; font-size: 14px; }
  button:disabled { opacity: 0.6; cursor: not-allowed; }
  .bb-auth-hint { color: #666; font-size: 13px; text-align: center; }
  .bb-auth-error { color: #e53e3e; font-size: 13px; text-align: center; }
</style>
