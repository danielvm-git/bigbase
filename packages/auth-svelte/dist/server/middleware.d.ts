import type { Handle, RequestEvent } from '@sveltejs/kit';
export interface SvelteKitAuthOptions {
    baseURL: string;
    cookieSecret: string;
    loginUrl?: string;
}
export declare function bigBaseAuthHandle(options: SvelteKitAuthOptions): Handle;
export declare function getSession(event: RequestEvent): any;
export declare function requireAuth(event: RequestEvent): Promise<any>;
