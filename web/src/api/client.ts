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

export interface TemplateRecord {
  id: string;
  tenant_id: string;
  name: string;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface TemplatesListResponse {
  count: number;
  templates: TemplateRecord[];
}

export interface WebhookConfigResponse {
  webhook_url: string;
  webhook_secret?: string;
}

export interface WebhookDelivery {
  id: number;
  tenant_id: string;
  notification_request_id: string;
  status_code: number;
  attempt: number;
  success: boolean;
  created_at: string;
}

export interface WebhookDeliveriesResponse {
  count: number;
  deliveries: WebhookDelivery[];
}

export interface DeadLetterListResponse {
  count: number;
  notifications: NotificationLog[];
}

export type ProviderHealthResponse = Record<string, string>;

export interface UsageResponse {
  tenant_id: string;
  plan: string;
  limit: number;
  used: number;
  remaining: number;
  resets_at: string;
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

// ── Auth & Tenant ─────────────────────────────────────────────────────────────
export async function registerTenant(name: string): Promise<ApiResponse<TenantRegisterResponse>> {
  return request<TenantRegisterResponse>('/v1/tenants', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });
}

// ── Notifications & Logs ──────────────────────────────────────────────────────
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

// ── Templates ─────────────────────────────────────────────────────────────────
export async function listTemplates(): Promise<ApiResponse<TemplatesListResponse>> {
  return request<TemplatesListResponse>('/v1/templates', { method: 'GET' });
}

export async function getTemplate(name: string): Promise<ApiResponse<TemplateRecord>> {
  return request<TemplateRecord>(`/v1/templates/${encodeURIComponent(name)}`, { method: 'GET' });
}

export async function createTemplate(name: string, body: string): Promise<ApiResponse<TemplateRecord>> {
  return request<TemplateRecord>('/v1/templates', {
    method: 'POST',
    body: JSON.stringify({ name, body }),
  });
}

export async function updateTemplate(name: string, body: string): Promise<ApiResponse<TemplateRecord>> {
  return request<TemplateRecord>(`/v1/templates/${encodeURIComponent(name)}`, {
    method: 'PUT',
    body: JSON.stringify({ body }),
  });
}

export async function deleteTemplate(name: string): Promise<ApiResponse<void>> {
  return request<void>(`/v1/templates/${encodeURIComponent(name)}`, { method: 'DELETE' });
}

// ── Webhooks ──────────────────────────────────────────────────────────────────
export async function getWebhookConfig(): Promise<ApiResponse<WebhookConfigResponse>> {
  return request<WebhookConfigResponse>('/v1/webhook', { method: 'GET' });
}

export async function setWebhook(webhookUrl: string): Promise<ApiResponse<WebhookConfigResponse>> {
  return request<WebhookConfigResponse>('/v1/webhook', {
    method: 'PUT',
    body: JSON.stringify({ webhook_url: webhookUrl }),
  });
}

export async function deleteWebhook(): Promise<ApiResponse<void>> {
  return request<void>('/v1/webhook', { method: 'DELETE' });
}

export async function getWebhookDeliveries(limit: number = 50): Promise<ApiResponse<WebhookDeliveriesResponse>> {
  return request<WebhookDeliveriesResponse>(`/v1/webhook/deliveries?limit=${limit}`, { method: 'GET' });
}

// ── Dead Letter Queue ─────────────────────────────────────────────────────────
export async function getDeadLetterNotifications(limit: number = 50): Promise<ApiResponse<DeadLetterListResponse>> {
  return request<DeadLetterListResponse>(`/v1/notify/dead-letter?limit=${limit}`, { method: 'GET' });
}

export async function replayNotification(requestId: string): Promise<ApiResponse<{ request_id: string; status: string }>> {
  return request<{ request_id: string; status: string }>(`/v1/notify/${encodeURIComponent(requestId)}/replay`, {
    method: 'POST',
  });
}

// ── Provider Health & Usage ───────────────────────────────────────────────────
export async function getProviderHealth(): Promise<ApiResponse<ProviderHealthResponse>> {
  return request<ProviderHealthResponse>('/v1/health/providers', { method: 'GET' });
}

export async function getTenantUsage(): Promise<ApiResponse<UsageResponse>> {
  return request<UsageResponse>('/v1/usage', { method: 'GET' });
}
