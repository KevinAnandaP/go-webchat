import { create } from 'zustand';
import { wsClient, WSMessage } from '@/lib/socket';
import { useChatStore } from './chatStore';
import { Message } from '@/lib/api';

interface SocketState {
  isConnected: boolean;
  isConnecting: boolean;
  
  // Actions
  connect: (token?: string) => Promise<void>;
  disconnect: () => void;
}

export const useSocketStore = create<SocketState>((set, get) => ({
  isConnected: false,
  isConnecting: false,

  connect: async (token) => {
    if (get().isConnected || get().isConnecting) return;
    
    set({ isConnecting: true });
    
    try {
      // Setup event handlers before connecting
      setupSocketHandlers();
      
      await wsClient.connect(token);
      set({ isConnected: true, isConnecting: false });
    } catch (error) {
      console.error('Failed to connect WebSocket:', error);
      set({ isConnected: false, isConnecting: false });
    }
  },

  disconnect: () => {
    wsClient.disconnect();
    set({ isConnected: false });
  },
}));

function setupSocketHandlers() {
  const chatStore = useChatStore.getState();

  // Connection status
  wsClient.onConnectionChange((connected) => {
    useSocketStore.setState({ isConnected: connected });
  });

  // New message
  wsClient.on('message:new', (msg: WSMessage) => {
    const message = msg.payload.message as Message;
    chatStore.addMessage(message);
    
    // Play notification sound if not in current conversation
    const currentConv = useChatStore.getState().currentConversation;
    if (currentConv?.id !== message.conversation_id) {
      playNotificationSound();
    }
  });

  // Message sent confirmation
  wsClient.on('message:sent', (msg: WSMessage) => {
    const { message_id, temp_id, status } = msg.payload as {
      message_id: string;
      temp_id: string;
      status: string;
    };
    // Update temp message with real ID
    const messages = useChatStore.getState().messages;
    const updatedMessages = messages.map((m) => {
      if (m.id === temp_id) {
        return { ...m, id: message_id, status: status as Message['status'] };
      }
      return m;
    });
    useChatStore.setState({ messages: updatedMessages });
  });

  // Message status update
  wsClient.on('message:status', (msg: WSMessage) => {
    const { message_id, status } = msg.payload as { message_id: string; status: Message['status'] };
    chatStore.updateMessageStatus(message_id, status);
  });

  // Typing indicators
  wsClient.on('user:typing', (msg: WSMessage) => {
    const { conversation_id, user_id } = msg.payload as { conversation_id: string; user_id: string };
    chatStore.setUserTyping(conversation_id, user_id, true);
  });

  wsClient.on('user:typing_stop', (msg: WSMessage) => {
    const { conversation_id, user_id } = msg.payload as { conversation_id: string; user_id: string };
    chatStore.setUserTyping(conversation_id, user_id, false);
  });

  // Online status
  wsClient.on('user:online', (msg: WSMessage) => {
    const { user_id } = msg.payload as { user_id: string };
    chatStore.setUserOnline(user_id, true);
  });

  wsClient.on('user:offline', (msg: WSMessage) => {
    const { user_id } = msg.payload as { user_id: string };
    chatStore.setUserOnline(user_id, false);
  });
}

function playNotificationSound() {
  try {
    const audio = new Audio('/notification.mp3');
    audio.volume = 0.5;
    audio.play().catch(() => {
      // Ignore autoplay errors
    });
  } catch {
    // Ignore errors
  }
}
