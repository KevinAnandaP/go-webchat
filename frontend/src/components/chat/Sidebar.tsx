'use client';

import React, { useState } from 'react';
import { Search, Plus, LogOut, Settings, Users, MessageCircle } from 'lucide-react';
import { Avatar } from '@/components/ui/Avatar';
import { ChatListSkeleton } from '@/components/ui/Skeleton';
import { AddContactModal } from '@/components/modals/AddContactModal';
import { CreateGroupModal } from '@/components/modals/CreateGroupModal';
import { useAuthStore } from '@/stores/authStore';
import { useChatStore } from '@/stores/chatStore';
import { cn, formatDate, truncate } from '@/lib/utils';
import { Conversation, conversationsApi } from '@/lib/api';

export function Sidebar() {
  const { user, logout } = useAuthStore();
  const { conversations, contacts, isLoadingConversations, currentConversation, setCurrentConversation, setMessages, fetchMessages } = useChatStore();
  
  const [searchQuery, setSearchQuery] = useState('');
  const [showAddContact, setShowAddContact] = useState(false);
  const [showCreateGroup, setShowCreateGroup] = useState(false);
  const [activeTab, setActiveTab] = useState<'chats' | 'contacts'>('chats');

  const filteredConversations = conversations.filter((conv) => {
    const name = conv.type === 'private' 
      ? conv.other_user?.name || ''
      : conv.group_name || '';
    return name.toLowerCase().includes(searchQuery.toLowerCase());
  });

  const filteredContacts = contacts.filter((contact) =>
    contact.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleSelectConversation = async (conv: Conversation) => {
    setCurrentConversation(conv);
    setMessages([]);
    await fetchMessages(conv.id);
  };

  const handleSelectContact = async (contactId: string) => {
    try {
      const response = await conversationsApi.create(contactId);
      setCurrentConversation(response.conversation);
      setMessages([]);
      await fetchMessages(response.conversation.id);
    } catch (error) {
      console.error('Failed to create conversation:', error);
    }
  };

  const handleLogout = async () => {
    await logout();
    window.location.href = '/login';
  };

  return (
    <>
      <aside className="w-80 h-full flex flex-col bg-[var(--bg-secondary)] border-r border-[var(--separator)]">
        {/* Header */}
        <div className="p-4 border-b border-[var(--separator)]">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <Avatar src={user?.avatar} name={user?.name || ''} size="md" isOnline={true} />
              <div>
                <h2 className="font-semibold text-[var(--text-primary)] text-sm">{user?.name}</h2>
                <p className="text-xs text-[var(--text-tertiary)]">{user?.unique_id}</p>
              </div>
            </div>
            <div className="flex items-center gap-1">
              <button
                onClick={() => setShowAddContact(true)}
                className="p-2 rounded-xl hover:bg-[var(--bg-tertiary)] transition-colors"
                title="Add Contact"
              >
                <Plus className="w-5 h-5 text-[var(--accent-blue)]" />
              </button>
              <button
                onClick={handleLogout}
                className="p-2 rounded-xl hover:bg-[var(--bg-tertiary)] transition-colors"
                title="Logout"
              >
                <LogOut className="w-5 h-5 text-[var(--text-tertiary)]" />
              </button>
            </div>
          </div>

          {/* Search */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-tertiary)]" />
            <input
              type="text"
              placeholder="Search..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full h-9 pl-9 pr-4 bg-[var(--bg-tertiary)] rounded-lg text-sm text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)] focus:outline-none focus:ring-2 focus:ring-[var(--accent-blue)]/20"
            />
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-[var(--separator)]">
          <button
            onClick={() => setActiveTab('chats')}
            className={cn(
              'flex-1 py-3 text-sm font-medium transition-colors',
              activeTab === 'chats'
                ? 'text-[var(--accent-blue)] border-b-2 border-[var(--accent-blue)]'
                : 'text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]'
            )}
          >
            <MessageCircle className="w-4 h-4 inline mr-1.5" />
            Chats
          </button>
          <button
            onClick={() => setActiveTab('contacts')}
            className={cn(
              'flex-1 py-3 text-sm font-medium transition-colors',
              activeTab === 'contacts'
                ? 'text-[var(--accent-blue)] border-b-2 border-[var(--accent-blue)]'
                : 'text-[var(--text-tertiary)] hover:text-[var(--text-secondary)]'
            )}
          >
            <Users className="w-4 h-4 inline mr-1.5" />
            Contacts
          </button>
        </div>

        {/* List */}
        <div className="flex-1 overflow-y-auto">
          {activeTab === 'chats' ? (
            isLoadingConversations ? (
              <ChatListSkeleton />
            ) : filteredConversations.length === 0 ? (
              <div className="p-8 text-center">
                <MessageCircle className="w-10 h-10 mx-auto text-[var(--text-quaternary)] mb-3" />
                <p className="text-sm text-[var(--text-tertiary)]">No conversations yet</p>
                <p className="text-xs text-[var(--text-quaternary)] mt-1">Add contacts to start chatting</p>
              </div>
            ) : (
              <div className="py-1">
                {filteredConversations.map((conv) => (
                  <ConversationItem
                    key={conv.id}
                    conversation={conv}
                    isActive={currentConversation?.id === conv.id}
                    onClick={() => handleSelectConversation(conv)}
                  />
                ))}
              </div>
            )
          ) : (
            <div className="py-1">
              <button
                onClick={() => setShowCreateGroup(true)}
                className="w-full flex items-center gap-3 px-4 py-3 hover:bg-[var(--bg-tertiary)] transition-colors"
              >
                <div className="w-12 h-12 rounded-full bg-[var(--accent-blue)] flex items-center justify-center">
                  <Users className="w-5 h-5 text-white" />
                </div>
                <span className="font-medium text-[var(--accent-blue)]">Create Group</span>
              </button>
              
              {filteredContacts.length === 0 ? (
                <div className="p-8 text-center">
                  <Users className="w-10 h-10 mx-auto text-[var(--text-quaternary)] mb-3" />
                  <p className="text-sm text-[var(--text-tertiary)]">No contacts yet</p>
                  <button
                    onClick={() => setShowAddContact(true)}
                    className="text-xs text-[var(--accent-blue)] mt-1 hover:underline"
                  >
                    Add your first contact
                  </button>
                </div>
              ) : (
                filteredContacts.map((contact) => (
                  <button
                    key={contact.id}
                    onClick={() => handleSelectContact(contact.id)}
                    className="w-full flex items-center gap-3 px-4 py-3 hover:bg-[var(--bg-tertiary)] transition-colors"
                  >
                    <Avatar src={contact.avatar} name={contact.name} size="lg" isOnline={contact.is_online} />
                    <div className="flex-1 text-left">
                      <h3 className="font-medium text-[var(--text-primary)]">{contact.name}</h3>
                      <p className="text-xs text-[var(--text-tertiary)]">{contact.unique_id}</p>
                    </div>
                  </button>
                ))
              )}
            </div>
          )}
        </div>
      </aside>

      <AddContactModal isOpen={showAddContact} onClose={() => setShowAddContact(false)} />
      <CreateGroupModal isOpen={showCreateGroup} onClose={() => setShowCreateGroup(false)} />
    </>
  );
}

function ConversationItem({ 
  conversation, 
  isActive, 
  onClick 
}: { 
  conversation: Conversation; 
  isActive: boolean; 
  onClick: () => void;
}) {
  const name = conversation.type === 'private' 
    ? conversation.other_user?.name || 'Unknown'
    : conversation.group_name || 'Group';
  
  const avatar = conversation.type === 'private'
    ? conversation.other_user?.avatar
    : conversation.group_icon;
    
  const isOnline = conversation.type === 'private' 
    ? conversation.other_user?.is_online 
    : undefined;

  const lastMessage = conversation.last_message?.content || 'No messages yet';
  const lastTime = conversation.last_message?.created_at 
    ? formatDate(conversation.last_message.created_at)
    : '';

  return (
    <button
      onClick={onClick}
      className={cn(
        'w-full flex items-center gap-3 px-4 py-3 transition-colors',
        isActive 
          ? 'bg-[var(--accent-blue)]/10' 
          : 'hover:bg-[var(--bg-tertiary)]'
      )}
    >
      <Avatar src={avatar} name={name} size="lg" isOnline={isOnline} />
      <div className="flex-1 min-w-0 text-left">
        <div className="flex items-center justify-between">
          <h3 className={cn(
            'font-medium truncate',
            isActive ? 'text-[var(--accent-blue)]' : 'text-[var(--text-primary)]'
          )}>
            {name}
          </h3>
          {lastTime && (
            <span className="text-xs text-[var(--text-tertiary)] ml-2 shrink-0">
              {lastTime}
            </span>
          )}
        </div>
        <p className="text-sm text-[var(--text-tertiary)] truncate">
          {truncate(lastMessage, 30)}
        </p>
      </div>
      {(conversation.unread_count ?? 0) > 0 && (
        <span className="shrink-0 min-w-5 h-5 px-1.5 flex items-center justify-center bg-[var(--accent-blue)] text-white text-xs font-medium rounded-full">
          {conversation.unread_count}
        </span>
      )}
    </button>
  );
}
