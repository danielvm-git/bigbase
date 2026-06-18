import { type NextRequest } from 'next/server';
export interface NextAuthConfig {
    baseURL: string;
    cookieSecret: string;
    loginUrl?: string;
}
export declare function bigBaseAuthMiddleware(config: NextAuthConfig): (request: NextRequest) => Promise<Response>;
export declare function authApiHandler(config: {
    baseURL: string;
    cookieSecret: string;
}): {
    GET: (request: Request, { params }: {
        params: {
            path: string[];
        };
    }) => Promise<Response>;
    POST: (request: Request, { params }: {
        params: {
            path: string[];
        };
    }) => Promise<Response>;
    PUT: (request: Request, { params }: {
        params: {
            path: string[];
        };
    }) => Promise<Response>;
    DELETE: (request: Request, { params }: {
        params: {
            path: string[];
        };
    }) => Promise<Response>;
    PATCH: (request: Request, { params }: {
        params: {
            path: string[];
        };
    }) => Promise<Response>;
};
