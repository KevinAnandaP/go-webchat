'use client';

import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Phone, Video, MoreVertical, ArrowLeft } from 'lucide-react';
import { Avatar } from '@/components/ui/Avatar';
import { MessageSkeleton } from '@/components/ui/Skeleton';
import { MessageBubble } from './MessageBubble';
import { MessageInput } from './MessageInput';
import { TypingIndicator } from './TypingIndicator';
import { useChatStore } from '@/stores/chatStore';
import { useAuthStore } from '@/stores/authStore';
import { wsClient } from '@/lib/socket';

export function ChatWindow() {
  const { user } = useAuthStore();
  const { 
    currentConversation, 
    setCurrentConversation,
    messages, 
    isLoadingMessages,
    fetchMessages,
    typingUsers 
  } = useChatStore();
  
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesContainerRef = useRef<HTMLDivElement>(null);
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  // Scroll to bottom on new messages
  useEffect(() => {
    if (!isLoadingMore) {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages, isLoadingMore]);

  // Infinite scroll - load more messages
  const handleScroll = useCallback(async () => {
    const container = messagesContainerRef.current;
    if (!container || !currentConversation || isLoadingMore || isLoadingMessages) return;

    if (container.scrollTop < 100) {
      setIsLoadingMore(true);
      const newMessages = await fetchMessages(currentConversation.id, messages.length);
      setIsLoadingMore(false);
      
      // Maintain scroll position
      if (newMessages.length > 0) {
        container.scrollTop = 200;
      }
    }
  }, [currentConversation, isLoadingMore, isLoadingMessages, messages.length, fetchMessages]);

  // Mark as read when opening conversation
  useEffect(() => {
    if (currentConversation) {
      wsClient.markAsRead(currentConversation.id);
    }
  }, [currentConversation]);

  if (!currentConversation) return null;

  const name = currentConversation.type === 'private' 
    ? currentConversation.other_user?.name || 'Unknown'
    : currentConversation.group_name || 'Group';
  
  const avatar = currentConversation.type === 'private'
    ? currentConversation.other_user?.avatar
    : currentConversation.group_icon;
    
  const isOnline = currentConversation.type === 'private' 
    ? currentConversation.other_user?.is_online 
    : false;

  const subtitle = currentConversation.type === 'private'
    ? isOnline ? 'Online' : 'Offline'
    : `${currentConversation.members?.length || 0} members`;

  // Get typing users for this conversation
  const conversationTypingUsers = typingUsers.get(currentConversation.id);
  const isTyping = conversationTypingUsers && conversationTypingUsers.size > 0;

  return (
    <div className="flex-1 flex flex-col h-full">
      {/* Header */}
      <header className="h-16 px-4 flex items-center gap-3 border-b border-[var(--separator)] bg-[var(--bg-elevated)] shrink-0">
        {/* Back button for mobile */}
        <button
          onClick={() => setCurrentConversation(null)}
          className="lg:hidden p-2 -ml-2 rounded-xl hover:bg-[var(--bg-secondary)] transition-colors"
        >
          <ArrowLeft className="w-5 h-5 text-[var(--text-tertiary)]" />
        </button>

        <Avatar src={avatar} name={name} size="md" isOnline={isOnline} />
        
        <div className="flex-1 min-w-0">
          <h2 className="font-semibold text-[var(--text-primary)] truncate">{name}</h2>
          <p className="text-xs text-[var(--text-tertiary)]">
            {isTyping ? 'typing...' : subtitle}
          </p>
        </div>

        <div className="flex items-center gap-1">
          <button className="p-2 rounded-xl hover:bg-[var(--bg-secondary)] transition-colors">
            <Phone className="w-5 h-5 text-[var(--text-tertiary)]" />
          </button>
          <button className="p-2 rounded-xl hover:bg-[var(--bg-secondary)] transition-colors">
            <Video className="w-5 h-5 text-[var(--text-tertiary)]" />
          </button>
          <button className="p-2 rounded-xl hover:bg-[var(--bg-secondary)] transition-colors">
            <MoreVertical className="w-5 h-5 text-[var(--text-tertiary)]" />
          </button>
        </div>
      </header>

      {/* Messages */}
      <div 
        ref={messagesContainerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto px-4 py-4"
      >
        {isLoadingMore && (
          <div className="py-4">
            <MessageSkeleton />
          </div>
        )}
        
        {isLoadingMessages ? (
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <MessageSkeleton key={i} />
            ))}
          </div>
        ) : messages.length === 0 ? (
          <div className="h-full flex items-center justify-center">
            <div className="text-center">
              <p className="text-[var(--text-tertiary)]">No messages yet</p>
              <p className="text-sm text-[var(--text-quaternary)] mt-1">
                Say hello to start the conversation!
              </p>
            </div>
          </div>
        ) : (
          <div className="space-y-1">
            {messages.map((message, index) => {
              const prevMessage = messages[index - 1];
              const showAvatar = !prevMessage || prevMessage.sender_id !== message.sender_id;
              const isOwn = message.sender_id === user?.id;

              return (
                <MessageBubble
                  key={message.id}
                  message={message}
                  isOwn={isOwn}
                  showAvatar={showAvatar && !isOwn}
                  isGroup={currentConversation.type === 'group'}
                />
              );
            })}
          </div>
        )}

        {isTyping && <TypingIndicator />}
        
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <MessageInput conversationId={currentConversation.id} />
    </div>
  );
}
