import React from 'react';
import { MessageCircle } from 'lucide-react';

export function EmptyState() {
  return (
    <div className="flex-1 flex items-center justify-center p-8">
      <div className="text-center max-w-sm animate-fadeIn">
        {/* Illustration */}
        <div className="relative mx-auto w-32 h-32 mb-6">
          <div className="absolute inset-0 bg-[var(--accent-blue)]/10 rounded-full animate-pulse" />
          <div className="absolute inset-4 bg-[var(--accent-blue)]/20 rounded-full" />
          <div className="absolute inset-0 flex items-center justify-center">
            <MessageCircle className="w-12 h-12 text-[var(--accent-blue)]" />
          </div>
        </div>

        <h2 className="text-xl font-semibold text-[var(--text-primary)] mb-2">
          Welcome to Go WebChat
        </h2>
        <p className="text-[var(--text-tertiary)] leading-relaxed">
          Select a conversation from the sidebar to start chatting, or add a new contact to begin a new conversation.
        </p>
      </div>
    </div>
  );
}
