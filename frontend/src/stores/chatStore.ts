import { create } from 'zustand';
import { Conversation, Message, User, conversationsApi, contactsApi } from '@/lib/api';

interface ChatState {
  // Data
  conversations: Conversation[];
  contacts: User[];
  currentConversation: Conversation | null;
  messages: Message[];
  
  // UI State
  isLoadingConversations: boolean;
  isLoadingMessages: boolean;
  isLoadingContacts: boolean;
  typingUsers: Map<string, Set<string>>; // conversationId -> Set of userIds
  onlineUsers: Set<string>;
  
  // Actions
  setConversations: (conversations: Conversation[]) => void;
  setContacts: (contacts: User[]) => void;
  setCurrentConversation: (conversation: Conversation | null) => void;
  setMessages: (messages: Message[]) => void;
  addMessage: (message: Message) => void;
  updateMessageStatus: (messageId: string, status: Message['status']) => void;
  
  // Typing
  setUserTyping: (conversationId: string, userId: string, isTyping: boolean) => void;
  
  // Online status
  setUserOnline: (userId: string, isOnline: boolean) => void;
  
  // Fetch actions
  fetchConversations: () => Promise<void>;
  fetchContacts: () => Promise<void>;
  fetchMessages: (conversationId: string, skip?: number) => Promise<Message[]>;
}

export const useChatStore = create<ChatState>((set, get) => ({
  conversations: [],
  contacts: [],
  currentConversation: null,
  messages: [],
  isLoadingConversations: false,
  isLoadingMessages: false,
  isLoadingContacts: false,
  typingUsers: new Map(),
  onlineUsers: new Set(),

  setConversations: (conversations) => set({ conversations }),
  
  setContacts: (contacts) => set({ contacts }),
  
  setCurrentConversation: (conversation) => set({ currentConversation: conversation }),
  
  setMessages: (messages) => set({ messages }),
  
  addMessage: (message) => {
    const { messages, conversations, currentConversation } = get();
    
    // Add to messages if in current conversation
    if (currentConversation?.id === message.conversation_id) {
      set({ messages: [...messages, message] });
    }
    
    // Update conversation's last message
    const updatedConversations = conversations.map((conv) => {
      if (conv.id === message.conversation_id) {
        return { ...conv, last_message: message, updated_at: message.created_at };
      }
      return conv;
    });
    
    // Sort by updated_at
    updatedConversations.sort(
      (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
    );
    
    set({ conversations: updatedConversations });
  },
  
  updateMessageStatus: (messageId, status) => {
    const { messages } = get();
    set({
      messages: messages.map((msg) =>
        msg.id === messageId ? { ...msg, status } : msg
      ),
    });
  },
  
  setUserTyping: (conversationId, userId, isTyping) => {
    const { typingUsers } = get();
    const newTypingUsers = new Map(typingUsers);
    
    if (!newTypingUsers.has(conversationId)) {
      newTypingUsers.set(conversationId, new Set());
    }
    
    const users = newTypingUsers.get(conversationId)!;
    if (isTyping) {
      users.add(userId);
    } else {
      users.delete(userId);
    }
    
    set({ typingUsers: newTypingUsers });
  },
  
  setUserOnline: (userId, isOnline) => {
    const { onlineUsers, contacts, conversations } = get();
    const newOnlineUsers = new Set(onlineUsers);
    
    if (isOnline) {
      newOnlineUsers.add(userId);
    } else {
      newOnlineUsers.delete(userId);
    }
    
    // Update contacts
    const updatedContacts = contacts.map((contact) => ({
      ...contact,
      is_online: newOnlineUsers.has(contact.id),
    }));
    
    // Update conversation other_user online status
    const updatedConversations = conversations.map((conv) => {
      if (conv.other_user && conv.other_user.id === userId) {
        return { ...conv, other_user: { ...conv.other_user, is_online: isOnline } };
      }
      return conv;
    });
    
    set({ onlineUsers: newOnlineUsers, contacts: updatedContacts, conversations: updatedConversations });
  },
  
  fetchConversations: async () => {
    set({ isLoadingConversations: true });
    try {
      const response = await conversationsApi.list();
      set({ conversations: response.conversations, isLoadingConversations: false });
    } catch {
      set({ isLoadingConversations: false });
    }
  },
  
  fetchContacts: async () => {
    set({ isLoadingContacts: true });
    try {
      const response = await contactsApi.list();
      set({ contacts: response.contacts, isLoadingContacts: false });
    } catch {
      set({ isLoadingContacts: false });
    }
  },
  
  fetchMessages: async (conversationId, skip = 0) => {
    set({ isLoadingMessages: skip === 0 });
    try {
      const response = await conversationsApi.messages(conversationId, skip);
      if (skip === 0) {
        set({ messages: response.messages, isLoadingMessages: false });
      } else {
        const { messages } = get();
        set({ messages: [...response.messages, ...messages], isLoadingMessages: false });
      }
      return response.messages;
    } catch {
      set({ isLoadingMessages: false });
      return [];
    }
  },
}));
