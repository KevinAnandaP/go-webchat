const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface ApiOptions extends RequestInit {
  data?: unknown;
}

class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
    this.name = 'ApiError';
  }
}

async function request<T>(endpoint: string, options: ApiOptions = {}): Promise<T> {
  const { data, ...customConfig } = options;

  const config: RequestInit = {
    method: data ? 'POST' : 'GET',
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
    },
    ...customConfig,
  };

  if (data) {
    config.body = JSON.stringify(data);
  }

  const response = await fetch(`${API_URL}${endpoint}`, config);

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new ApiError(error.error || 'Request failed', response.status);
  }

  return response.json();
}

export const api = {
  get: <T>(endpoint: string, options?: ApiOptions) =>
    request<T>(endpoint, { ...options, method: 'GET' }),

  post: <T>(endpoint: string, data?: unknown, options?: ApiOptions) =>
    request<T>(endpoint, { ...options, method: 'POST', data }),

  put: <T>(endpoint: string, data?: unknown, options?: ApiOptions) =>
    request<T>(endpoint, { ...options, method: 'PUT', data }),

  delete: <T>(endpoint: string, options?: ApiOptions) =>
    request<T>(endpoint, { ...options, method: 'DELETE' }),
};

// Auth API
export const authApi = {
  register: (data: { email: string; password: string; name: string }) =>
    api.post<{ message: string; user: User }>('/api/auth/register', data),

  login: (data: { email: string; password: string; remember_me?: boolean }) =>
    api.post<{ message: string; user: User }>('/api/auth/login', data),

  logout: () => api.post<{ message: string }>('/api/auth/logout'),

  me: () => api.get<{ user: User }>('/api/auth/me'),
};

// Contacts API
export const contactsApi = {
  list: () => api.get<{ contacts: User[] }>('/api/contacts'),

  add: (uniqueId: string) =>
    api.post<{ message: string; contact: User }>('/api/contacts', { unique_id: uniqueId }),

  remove: (id: string) => api.delete<{ message: string }>(`/api/contacts/${id}`),

  search: (uniqueId: string) => api.get<{ user: User }>(`/api/contacts/search?unique_id=${uniqueId}`),
};

// Conversations API
export const conversationsApi = {
  list: () => api.get<{ conversations: Conversation[] }>('/api/conversations'),

  create: (userId: string) =>
    api.post<{ conversation: Conversation }>('/api/conversations', { user_id: userId }),

  get: (id: string) => api.get<{ conversation: Conversation }>(`/api/conversations/${id}`),

  messages: (id: string, skip = 0, limit = 50) =>
    api.get<{ messages: Message[] }>(`/api/conversations/${id}/messages?skip=${skip}&limit=${limit}`),
};

// Groups API
export const groupsApi = {
  create: (data: { name: string; icon?: string; member_ids: string[] }) =>
    api.post<{ group: Conversation }>('/api/groups', data),

  update: (id: string, data: { name?: string; icon?: string }) =>
    api.put<{ message: string; group: Conversation }>(`/api/groups/${id}`, data),

  addMember: (id: string, userId: string) =>
    api.post<{ message: string }>(`/api/groups/${id}/members`, { user_id: userId }),

  removeMember: (id: string, userId: string) =>
    api.delete<{ message: string }>(`/api/groups/${id}/members/${userId}`),

  leave: (id: string) => api.post<{ message: string }>(`/api/groups/${id}/leave`),
};

// Types
export interface User {
  id: string;
  unique_id: string;
  name: string;
  avatar: string;
  last_seen?: string;
  is_online?: boolean;
}

export interface Conversation {
  id: string;
  type: 'private' | 'group';
  members: string[];
  group_name?: string;
  group_icon?: string;
  admin?: string;
  created_at: string;
  updated_at: string;
  last_message?: Message;
  other_user?: User;
  members_list?: User[];
  unread_count?: number;
}

export interface Message {
  id: string;
  conversation_id: string;
  sender_id: string;
  content: string;
  status: 'sent' | 'delivered' | 'read';
  read_by: string[];
  created_at: string;
  sender?: User;
}
