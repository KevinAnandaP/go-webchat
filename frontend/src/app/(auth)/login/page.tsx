'use client';

import React, { useState, Suspense } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { useAuthStore } from '@/stores/authStore';

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirect = searchParams.get('redirect') || '/chat';
  
  const { login } = useAuthStore();
  
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      await login(email, password, rememberMe);
      router.push(redirect);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Login failed';
      setError(message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleGoogleLogin = () => {
    window.location.href = `${process.env.NEXT_PUBLIC_API_URL}/api/auth/google`;
  };

  return (
    <div className="w-full max-w-sm animate-slideUp">
      <div className="text-center mb-8">
        <h1 className="text-2xl font-bold text-[var(--text-primary)]">Welcome Back</h1>
        <p className="text-[var(--text-tertiary)] mt-2">Sign in to your account</p>
      </div>

      <div className="bg-[var(--bg-elevated)] rounded-2xl p-8 shadow-sm border border-[var(--border-light)]">
        <form onSubmit={handleSubmit} className="space-y-5">
          {error && (
            <div className="p-3 text-sm text-[var(--accent-red)] bg-[var(--accent-red)]/10 rounded-lg">
              {error}
            </div>
          )}

          <Input
            type="email"
            label="Email"
            placeholder="name@example.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            className="bg-[var(--bg-secondary)]"
          />

          <Input
            type="password"
            label="Password"
            placeholder="Your password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            className="bg-[var(--bg-secondary)]"
          />

          <div className="flex items-center">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                className="w-4 h-4 rounded border-[var(--text-quaternary)] text-[var(--accent-blue)] focus:ring-[var(--accent-blue)]"
              />
              <span className="text-sm text-[var(--text-secondary)]">Remember me</span>
            </label>
          </div>

          <Button
            type="submit"
            className="w-full font-semibold"
            size="lg"
            isLoading={isLoading}
          >
            Sign In
          </Button>
        </form>

        <div className="relative my-8">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-[var(--separator)]" />
          </div>
          <div className="relative flex justify-center text-xs uppercase tracking-wider font-medium">
            <span className="px-3 bg-[var(--bg-elevated)] text-[var(--text-tertiary)]">
              Or
            </span>
          </div>
        </div>

        <Button
          type="button"
          variant="secondary"
          className="w-full font-medium"
          onClick={handleGoogleLogin}
        >
          Continue with Google
        </Button>
        <p className="text-center text-sm text-[var(--text-secondary)] mt-6 pt-4 border-t border-[var(--separator)]">
          Don&apos;t have an account?{' '}
          <Link
            href="/register"
            className="text-[var(--accent-blue)] font-medium hover:underline"
          >
            Sign up
          </Link>
        </p>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-[var(--bg-secondary)]">
      <Suspense fallback={
        <div className="w-full max-w-sm animate-pulse">
          <div className="h-16 w-16 mx-auto rounded-2xl bg-[var(--bg-tertiary)] mb-4" />
          <div className="h-8 w-48 mx-auto rounded bg-[var(--bg-tertiary)] mb-8" />
          <div className="bg-[var(--bg-elevated)] rounded-2xl p-6 space-y-4">
            <div className="h-11 rounded-xl bg-[var(--bg-tertiary)]" />
            <div className="h-11 rounded-xl bg-[var(--bg-tertiary)]" />
            <div className="h-11 rounded-xl bg-[var(--bg-tertiary)]" />
          </div>
        </div>
      }>
        <LoginForm />
      </Suspense>
    </div>
  );
}
