// NOTE ON AUTH TRADEOFF:
// Storing the API key in localStorage is acceptable here because RelayHub Dashboard is a developer
// testing/internal engineering tool designed for local inspection, not a production public auth client.

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
const STORAGE_KEY = 'relayhub_api_key';

let inMemoryApiKey: string | null = localStorage.getItem(STORAGE_KEY);

export function getApiKey(): string | null {
  return inMemoryApiKey;
}

export function setApiKey(key: string): void {
  inMemoryApiKey = key;
  localStorage.setItem(STORAGE_KEY, key);
}

export function clearApiKey(): void {
  inMemoryApiKey = null;
  localStorage.removeItem(STORAGE_KEY);
}

export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  missing_variables?: string[];
}

export interface TenantRegisterResponse {
  tenant_id: string;
  api_key: string;
}

export interface SendNotifyRequest {
  channel: 'email' | 'discord' | 'smtp' | 'auto';
  recipient?: string;
  discord_recipient?: string;
  email_recipient?: string;
  message?: string;
  template?: string;
  variables?: Record<string, string>;
  idempotency_key?: string;
  send_at?: string;
}

export interface NotificationResponse {
  request_id: string;
  status: string;
  channel?: string;
  scheduled_for?: string;
}

export interface NotificationLog {
  id: number;
  tenant_id: string;
  request_id: string;
  recipient: string;
  channel: string;
  message: string;
  status: 'queued' | 'delivered' | 'failed' | 'scheduled' | 'dead_letter' | 'cancelled';
  error_message: string;
  attempts: number;
  fallback_used: boolean;
  idempotency_key: string;
  created_at: string;
  scheduled_for?: string;
  discord_recipient?: string;
  email_recipient?: string;
}

export interface LogsResponse {
  count: number;
  logs: NotificationLog[];
}

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
  const url = `${API_BASE_URL}${endpoint}`;
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  };

  const apiKey = getApiKey();
  if (apiKey) {
    headers['X-API-Key'] = apiKey;
  }

  try {
    const res = await fetch(url, {
      ...options,
      headers,
    });

    const data = await res.json();
    if (!res.ok) {
      return {
        success: false,
        error: data.error || `HTTP ${res.status}: ${res.statusText}`,
        missing_variables: data.missing_variables,
      };
    }

    return data;
  } catch (err: any) {
    return {
      success: false,
      error: err.message || 'Network error — is the RelayHub server running?',
    };
  }
}

export async function registerTenant(name: string): Promise<ApiResponse<TenantRegisterResponse>> {
  return request<TenantRegisterResponse>('/v1/tenants', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });
}

export async function sendNotification(req: SendNotifyRequest): Promise<ApiResponse<NotificationResponse>> {
  return request<NotificationResponse>('/v1/notify', {
    method: 'POST',
    body: JSON.stringify(req),
  });
}

export async function getNotificationStatus(requestId: string): Promise<ApiResponse<NotificationLog>> {
  return request<NotificationLog>(`/v1/notify/${requestId}`, {
    method: 'GET',
  });
}

export async function getLogs(limit: number = 50): Promise<ApiResponse<LogsResponse>> {
  return request<LogsResponse>(`/v1/logs?limit=${limit}`, {
    method: 'GET',
  });
}
