declare module 'svelte/store' {
  export function writable<T>(v: T): { subscribe(fn: (v: T) => void): () => void; set(v: T): void; update(fn: (v: T) => T): T };
  export function derived<S, T>(store: any, fn: (v: S) => T): { subscribe(fn: (v: T) => void): () => void };
}
declare module '$app/environment' {
  export const browser: boolean;
}
declare module '@sveltejs/kit' {
  export interface RequestEvent {
    cookies: { get(name: string): string | undefined; delete(name: string, opts: any): void };
    url: URL;
    locals: any;
  }
  export interface Handle {
    (input: { event: RequestEvent; resolve: (event: RequestEvent) => Response | Promise<Response> }): Response | Promise<Response>;
  }
}
