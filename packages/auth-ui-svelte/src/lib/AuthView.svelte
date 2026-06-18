<script lang="ts">
  import type { AuthClient } from '@bigbase/auth';
  import SignIn from './SignIn.svelte';
  import SignUp from './SignUp.svelte';

  interface Props {
    authClient: AuthClient;
    path?: 'sign-in' | 'sign-up';
    redirectTo?: string;
  }

  let { authClient, path = 'sign-in', redirectTo = '/' }: Props = $props();

  let currentPath = $state(path);

  function switchToSignUp() { currentPath = 'sign-up'; }
  function switchToSignIn() { currentPath = 'sign-in'; }
</script>

<div class="bb-auth-view">
  {#if currentPath === 'sign-in'}
    <SignIn {authClient} {redirectTo} />
    <p class="bb-auth-switch">
      Don't have an account? <button onclick={switchToSignUp}>Sign up</button>
    </p>
  {:else}
    <SignUp {authClient} {redirectTo} />
    <p class="bb-auth-switch">
      Already have an account? <button onclick={switchToSignIn}>Sign in</button>
    </p>
  {/if}
</div>

<style>
  .bb-auth-view { max-width: 400px; margin: 40px auto; }
  .bb-auth-switch { text-align: center; margin-top: 16px; font-size: 14px; color: #666; }
  .bb-auth-switch button { background: none; border: none; color: #1a1a2e; cursor: pointer; font-weight: 500; text-decoration: underline; }
</style>
