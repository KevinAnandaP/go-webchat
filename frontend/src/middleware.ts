import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// Routes yang memerlukan authentication
const protectedRoutes = ['/chat'];

// Routes yang hanya untuk guest (belum login)
const guestRoutes = ['/login', '/register'];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  
  // Check for auth token in cookies
  const authToken = request.cookies.get('auth_token')?.value;
  const isAuthenticated = !!authToken;

  // Redirect authenticated users away from guest routes
  if (isAuthenticated && guestRoutes.some(route => pathname.startsWith(route))) {
    return NextResponse.redirect(new URL('/chat', request.url));
  }

  // Redirect unauthenticated users to login
  if (!isAuthenticated && protectedRoutes.some(route => pathname.startsWith(route))) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/((?!api|_next/static|_next/image|favicon.ico|.*\\..*).*)' 
  ],
};
