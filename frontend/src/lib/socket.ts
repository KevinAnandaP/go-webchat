import { Message } from './api';

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws';

export type WSEventType =
  | 'message:new'
  | 'message:sent'
  | 'message:status'
  | 'user:typing'
  | 'user:typing_stop'
  | 'user:online'
  | 'user:offline'
  | 'group:created'
  | 'group:updated'
  | 'group:member_added'
  | 'group:member_removed'
  | 'pong'
  | 'error';

export interface WSMessage {
  type: WSEventType;
  payload: Record<string, unknown>;
}

type MessageHandler = (message: WSMessage) => void;

class WebSocketClient {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;
  private pingInterval: NodeJS.Timeout | null = null;
  private handlers: Map<WSEventType, Set<MessageHandler>> = new Map();
  private connectionHandlers: Set<(connected: boolean) => void> = new Set();

  connect(token?: string): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      const url = token ? `${WS_URL}?token=${token}` : WS_URL;
      this.ws = new WebSocket(url);

      this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.reconnectAttempts = 0;
        this.startPing();
        this.notifyConnection(true);
        resolve();
      };

      this.ws.onclose = () => {
        console.log('WebSocket disconnected');
        this.stopPing();
        this.notifyConnection(false);
        this.attemptReconnect();
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        reject(error);
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };
    });
  }

  disconnect() {
    this.stopPing();
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private startPing() {
    this.pingInterval = setInterval(() => {
      this.send({ type: 'ping', payload: {} });
    }, 30000);
  }

  private stopPing() {
    if (this.pingInterval) {
      clearInterval(this.pingInterval);
      this.pingInterval = null;
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
      console.log(`Attempting to reconnect in ${delay}ms...`);
      setTimeout(() => this.connect(), delay);
    }
  }

  private handleMessage(message: WSMessage) {
    const handlers = this.handlers.get(message.type);
    if (handlers) {
      handlers.forEach((handler) => handler(message));
    }
  }

  private notifyConnection(connected: boolean) {
    this.connectionHandlers.forEach((handler) => handler(connected));
  }

  send(message: { type: string; payload: Record<string, unknown> }) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  // Event handlers
  on(event: WSEventType, handler: MessageHandler) {
    if (!this.handlers.has(event)) {
      this.handlers.set(event, new Set());
    }
    this.handlers.get(event)!.add(handler);
    return () => this.off(event, handler);
  }

  off(event: WSEventType, handler: MessageHandler) {
    this.handlers.get(event)?.delete(handler);
  }

  onConnectionChange(handler: (connected: boolean) => void) {
    this.connectionHandlers.add(handler);
    return () => this.connectionHandlers.delete(handler);
  }

  // Convenience methods
  sendMessage(conversationId: string, content: string, tempId?: string) {
    this.send({
      type: 'message:send',
      payload: { conversation_id: conversationId, content, temp_id: tempId },
    });
  }

  startTyping(conversationId: string) {
    this.send({
      type: 'typing:start',
      payload: { conversation_id: conversationId },
    });
  }

  stopTyping(conversationId: string) {
    this.send({
      type: 'typing:stop',
      payload: { conversation_id: conversationId },
    });
  }

  markAsRead(conversationId: string, messageId?: string) {
    this.send({
      type: 'message:read',
      payload: { conversation_id: conversationId, message_id: messageId },
    });
  }

  get isConnected() {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

export const wsClient = new WebSocketClient();
