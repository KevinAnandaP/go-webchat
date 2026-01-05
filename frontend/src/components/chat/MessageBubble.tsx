import React from 'react';
import { Check, CheckCheck } from 'lucide-react';
import { Avatar } from '@/components/ui/Avatar';
import { Message } from '@/lib/api';
import { cn, formatTime } from '@/lib/utils';

interface MessageBubbleProps {
  message: Message;
  isOwn: boolean;
  showAvatar: boolean;
  isGroup: boolean;
}

export function MessageBubble({ message, isOwn, showAvatar, isGroup }: MessageBubbleProps) {
  const statusIcon = () => {
    if (!isOwn) return null;
    
    switch (message.status) {
      case 'sent':
        return <Check className="w-3.5 h-3.5" />;
      case 'delivered':
        return <CheckCheck className="w-3.5 h-3.5" />;
      case 'read':
        return <CheckCheck className="w-3.5 h-3.5 text-[var(--accent-blue)]" />;
      default:
        return null;
    }
  };

  return (
    <div
      className={cn(
        'flex gap-2 max-w-[85%] animate-slideUp',
        isOwn ? 'ml-auto flex-row-reverse' : 'mr-auto'
      )}
    >
      {/* Avatar */}
      {!isOwn && (
        <div className="w-8 shrink-0">
          {showAvatar && message.sender && (
            <Avatar src={message.sender.avatar} name={message.sender.name} size="sm" />
          )}
        </div>
      )}

      {/* Bubble */}
      <div className="flex flex-col">
        {/* Sender name (for groups) */}
        {isGroup && !isOwn && showAvatar && message.sender && (
          <span className="text-xs text-[var(--accent-blue)] font-medium mb-0.5 ml-3">
            {message.sender.name}
          </span>
        )}

        <div
          className={cn(
            'px-3.5 py-2 rounded-2xl',
            isOwn
              ? 'bg-[var(--bubble-sent)] text-[var(--bubble-sent-text)] rounded-br-md'
              : 'bg-[var(--bubble-received)] text-[var(--bubble-received-text)] rounded-bl-md'
          )}
        >
          <p className="text-[15px] leading-relaxed whitespace-pre-wrap break-words">
            {message.content}
          </p>
        </div>

        {/* Time & Status */}
        <div
          className={cn(
            'flex items-center gap-1 mt-0.5 text-[11px] text-[var(--text-tertiary)]',
            isOwn ? 'justify-end mr-1' : 'ml-3'
          )}
        >
          <span>{formatTime(message.created_at)}</span>
          {statusIcon()}
        </div>
      </div>
    </div>
  );
}
