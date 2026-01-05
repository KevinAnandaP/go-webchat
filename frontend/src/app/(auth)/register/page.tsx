'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { useAuthStore } from '@/stores/authStore';

export default function RegisterPage() {
  const router = useRouter();
  const { register } = useAuthStore();
  
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    if (password.length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }

    setIsLoading(true);

    try {
      await register(email, password, name);
      router.push('/chat');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Registration failed';
      setError(message);
    } finally {
      setIsLoading(false);
    }
  };

  const handleGoogleLogin = () => {
    window.location.href = `${process.env.NEXT_PUBLIC_API_URL}/api/auth/google`;
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-[var(--bg-secondary)]">
      <div className="w-full max-w-sm animate-slideUp">
        <div className="text-center mb-8">
          <h1 className="text-2xl font-bold text-[var(--text-primary)]">Create Account</h1>
          <p className="text-[var(--text-tertiary)] mt-2">Sign up to get started</p>
        </div>

        <div className="bg-[var(--bg-elevated)] rounded-2xl p-8 shadow-sm border border-[var(--border-light)]">
          <form onSubmit={handleSubmit} className="space-y-5">
            {error && (
              <div className="p-3 text-sm text-[var(--accent-red)] bg-[var(--accent-red)]/10 rounded-lg">
                {error}
              </div>
            )}

            <Input
              type="text"
              label="Full Name"
              placeholder="John Doe"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="bg-[var(--bg-secondary)]"
            />

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
              placeholder="Create a password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              className="bg-[var(--bg-secondary)]"
            />

            <Input
              type="password"
              label="Confirm Password"
              placeholder="Confirm your password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              error={confirmPassword && password !== confirmPassword ? 'Passwords do not match' : undefined}
              required
              className="bg-[var(--bg-secondary)]"
            />

            <Button
              type="submit"
              className="w-full font-semibold"
              size="lg"
              isLoading={isLoading}
            >
              Create Account
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
            Sign up with Google
          </Button>
        </div>

        <p className="text-center text-sm text-[var(--text-secondary)] mt-8">
          Already have an account?{' '}
          <Link
            href="/login"
            className="text-[var(--accent-blue)] font-medium hover:underline"
          >
            Sign in
          </Link>
        </p>
      </div>
    </div>
  );
}
