declare module 'next/server' {
  export class NextResponse {
    static next(opts?: any): Response;
    static redirect(url: URL | string, init?: any): Response;
  }
  export interface NextRequest {
    nextUrl: URL;
    url: string;
    cookies: { get(name: string): { value: string } | undefined; delete(name: string): void };
    headers: Headers;
  }
}
