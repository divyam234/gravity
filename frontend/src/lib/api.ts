import type { ApiResponse, Download, Provider, Remote, Stats, MagnetInfo, MagnetDownloadRequest, DownloadFile } from './types';

export type { ApiResponse };

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

class ApiClient {
  private apiKey: string = '';
  private baseUrl: string = '/api/v1';

  setApiKey(key: string) {
    this.apiKey = key;
  }

  setBaseUrl(url: string) {
    this.baseUrl = url.replace(/\/+$/, "");
  }

  public async request<T>(method: string, path: string, body?: any): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.apiKey) {
      headers['X-API-Key'] = this.apiKey;
    }

    const response = await fetch(`${this.baseUrl}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new ApiError(response.status, errorText || response.statusText);
    }

    if (response.status === 204) {
      return {} as T;
    }

    return response.json();
  }

  // Downloads
  getDownloads(params?: { status?: string[]; limit?: number; offset?: number }) {
    let query = '';
    if (params) {
      const q = new URLSearchParams();
      if (params.status) q.append('status', params.status.join(','));
      if (params.limit) q.append('limit', params.limit.toString());
      if (params.offset) q.append('offset', params.offset.toString());
      query = `?${q.toString()}`;
    }
    return this.request<ApiResponse<Download[]>>('GET', `/downloads${query}`);
  }

  getDownload(id: string) {
    return this.request<Download>('GET', `/downloads/${id}`);
  }

  createDownload(url: string, destination?: string, filename?: string) {
    return this.request<Download>('POST', '/downloads', { url, destination, filename });
  }

  pauseDownload(id: string) {
    return this.request<void>('POST', `/downloads/${id}/pause`);
  }

  resumeDownload(id: string) {
    return this.request<void>('POST', `/downloads/${id}/resume`);
  }

  deleteDownload(id: string, deleteFiles = false) {
    return this.request<void>('DELETE', `/downloads/${id}?deleteFiles=${deleteFiles}`);
  }

  // Providers
  getProviders() {
    return this.request<ApiResponse<Provider[]>>('GET', '/providers');
  }

  configureProvider(name: string, config: Record<string, string>, enabled: boolean) {
    return this.request<void>('PUT', `/providers/${name}`, { config, enabled });
  }

  resolveUrl(url: string) {
    return this.request<any>('POST', '/providers/resolve', { url });
  }

  // Remotes
  getRemotes() {
    return this.request<ApiResponse<Remote[]>>('GET', '/remotes');
  }

  // Stats
  getStats() {
    return this.request<Stats>('GET', '/stats');
  }

  // Settings
  getSettings() {
    return this.request<Record<string, string>>('GET', '/settings');
  }

  updateSettings(settings: Record<string, string>) {
    return this.request<void>('PATCH', '/settings', settings);
  }

  // System
  getVersion() {
    return this.request<{ version: string; aria2: string; rclone: string }>('GET', '/system/version');
  }

  // Magnets
  checkMagnet(magnet: string) {
    return this.request<MagnetInfo>('POST', '/magnets/check', { magnet });
  }

  checkTorrent(torrentBase64: string) {
    return this.request<MagnetInfo>('POST', '/magnets/check-torrent', { torrentBase64 });
  }

  downloadMagnet(req: MagnetDownloadRequest) {
    return this.request<Download>('POST', '/magnets/download', req);
  }

  getDownloadFiles(id: string) {
    return this.request<{ files: DownloadFile[] }>('GET', `/downloads/${id}/files`);
  }
}

export const api = new ApiClient();
