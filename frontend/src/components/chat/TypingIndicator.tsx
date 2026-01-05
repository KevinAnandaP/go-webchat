import React from 'react';

export function TypingIndicator() {
  return (
    <div className="flex items-center gap-2 ml-10 my-2 animate-fadeIn">
      <div className="flex items-center gap-1 px-3 py-2 bg-[var(--bubble-received)] rounded-2xl rounded-bl-md">
        <span className="w-2 h-2 bg-[var(--text-tertiary)] rounded-full typing-dot" />
        <span className="w-2 h-2 bg-[var(--text-tertiary)] rounded-full typing-dot" />
        <span className="w-2 h-2 bg-[var(--text-tertiary)] rounded-full typing-dot" />
      </div>
    </div>
  );
}
