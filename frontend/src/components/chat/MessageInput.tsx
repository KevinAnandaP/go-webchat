'use client';

import React, { useState, useRef, useEffect } from 'react';
import { Send, Smile, Paperclip } from 'lucide-react';
import { wsClient } from '@/lib/socket';
import { useChatStore } from '@/stores/chatStore';
import { useAuthStore } from '@/stores/authStore';
import { cn } from '@/lib/utils';

interface MessageInputProps {
  conversationId: string;
}

export function MessageInput({ conversationId }: MessageInputProps) {
  const { user } = useAuthStore();
  const { addMessage } = useChatStore();
  const [message, setMessage] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

  // Auto-resize textarea
  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.style.height = 'auto';
      inputRef.current.style.height = `${Math.min(inputRef.current.scrollHeight, 120)}px`;
    }
  }, [message]);

  const handleTyping = () => {
    if (!isTyping) {
      setIsTyping(true);
      wsClient.startTyping(conversationId);
    }

    // Clear existing timeout
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }

    // Set new timeout to stop typing
    typingTimeoutRef.current = setTimeout(() => {
      setIsTyping(false);
      wsClient.stopTyping(conversationId);
    }, 2000);
  };

  const handleSend = () => {
    const content = message.trim();
    if (!content || !user) return;

    // Clear typing
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }
    if (isTyping) {
      wsClient.stopTyping(conversationId);
      setIsTyping(false);
    }

    // Generate temp ID for optimistic UI
    const tempId = `temp-${Date.now()}`;

    // Optimistic update
    addMessage({
      id: tempId,
      conversation_id: conversationId,
      sender_id: user.id,
      content,
      status: 'sent',
      read_by: [user.id],
      created_at: new Date().toISOString(),
      sender: user,
    });

    // Send via WebSocket
    wsClient.sendMessage(conversationId, content, tempId);

    // Clear input
    setMessage('');
    inputRef.current?.focus();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="p-3 border-t border-[var(--separator)] bg-[var(--bg-elevated)]">
      <div className="flex items-end gap-2">
        {/* Attachment button */}
        <button className="p-2.5 rounded-xl hover:bg-[var(--bg-secondary)] transition-colors shrink-0">
          <Paperclip className="w-5 h-5 text-[var(--text-tertiary)]" />
        </button>

        {/* Input container */}
        <div className="flex-1 relative">
          <textarea
            ref={inputRef}
            value={message}
            onChange={(e) => {
              setMessage(e.target.value);
              handleTyping();
            }}
            onKeyDown={handleKeyDown}
            placeholder="Type a message..."
            rows={1}
            className={cn(
              'w-full px-4 py-2.5 pr-10',
              'bg-[var(--bg-secondary)] rounded-2xl',
              'text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)]',
              'resize-none overflow-hidden',
              'focus:outline-none focus:ring-2 focus:ring-[var(--accent-blue)]/20',
              'transition-all'
            )}
          />
          <button className="absolute right-2 bottom-2 p-1.5 rounded-lg hover:bg-[var(--bg-tertiary)] transition-colors">
            <Smile className="w-5 h-5 text-[var(--text-tertiary)]" />
          </button>
        </div>

        {/* Send button */}
        <button
          onClick={handleSend}
          disabled={!message.trim()}
          className={cn(
            'p-2.5 rounded-xl transition-all shrink-0',
            message.trim()
              ? 'bg-[var(--accent-blue)] text-white hover:bg-[#0066DD] active:scale-95'
              : 'bg-[var(--bg-secondary)] text-[var(--text-tertiary)]'
          )}
        >
          <Send className="w-5 h-5" />
        </button>
      </div>
    </div>
  );
}
