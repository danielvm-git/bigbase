<script lang="ts">
  import type { AuthClient, AuthUser } from '@bigbase/auth';
  import { onMount } from 'svelte';

  interface Props {
    authClient: AuthClient;
    loginUrl?: string;
  }

  let { authClient, loginUrl = '/login' }: Props = $props();

  let user = $state<AuthUser | null>(null);
  let open = $state(false);

  onMount(() => {
    authClient.getSession().then(s => user = s?.user ?? null);
    return authClient.onAuthStateChange(s => user = s?.user ?? null);
  });

  function toggle() { open = !open; }
  function close() { open = false; }

  async function handleSignOut() {
    await authClient.signOut();
    window.location.href = loginUrl;
  }

  function getInitials(name?: string) {
    if (!name) return '?';
    return name.split(' ').slice(0, 2).map(w => w[0]).join('').toUpperCase();
  }
</script>

{#if user}
  <div class="bb-auth-userbtn">
    <button class="bb-auth-avatar" onclick={toggle} aria-label="User menu">
      {#if user.avatar_url}
        <img src={user.avatar_url} alt={user.name || user.email} />
      {:else}
        <span class="bb-auth-initials">{getInitials(user.name)}</span>
      {/if}
    </button>
    {#if open}
      <!-- svelte-ignore a11y-click-events-have-key-events -->
      <!-- svelte-ignore a11y-no-static-element-interactions -->
      <div class="bb-auth-backdrop" onclick={close}></div>
      <div class="bb-auth-dropdown">
        <p class="bb-auth-dropdown-name">{user.name || user.email}</p>
        <p class="bb-auth-dropdown-email">{user.email}</p>
        <button class="bb-auth-signout" onclick={handleSignOut}>Sign out</button>
      </div>
    {/if}
  </div>
{:else}
  <a href={loginUrl} class="bb-auth-signin-link">Sign in</a>
{/if}

<style>
  .bb-auth-userbtn { position: relative; display: inline-block; }
  .bb-auth-avatar { width: 36px; height: 36px; border-radius: 50%; border: none; cursor: pointer; overflow: hidden; padding: 0; }
  .bb-auth-avatar img { width: 100%; height: 100%; object-fit: cover; }
  .bb-auth-initials { display: flex; align-items: center; justify-content: center; width: 100%; height: 100%; background: #1a1a2e; color: white; font-size: 12px; font-weight: 600; }
  .bb-auth-backdrop { position: fixed; inset: 0; z-index: 99; }
  .bb-auth-dropdown { position: absolute; right: 0; top: 44px; background: white; border: 1px solid #e2e8f0; border-radius: 8px; padding: 12px; min-width: 200px; z-index: 100; box-shadow: 0 4px 12px rgba(0,0,0,0.1); }
  .bb-auth-dropdown-name { font-weight: 600; margin: 0 0 4px; }
  .bb-auth-dropdown-email { color: #666; font-size: 13px; margin: 0 0 8px; }
  .bb-auth-signout { width: 100%; padding: 6px; background: none; border: 1px solid #e2e8f0; border-radius: 4px; cursor: pointer; font-size: 13px; }
  .bb-auth-signout:hover { background: #f7fafc; }
  .bb-auth-signin-link { color: #1a1a2e; text-decoration: none; font-weight: 500; }
</style>
