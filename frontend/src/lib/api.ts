import type {
  ApiResponse,
  Download,
  Provider,
  Remote,
  Stats,
  MagnetInfo,
  MagnetDownloadRequest,
  FileInfo,
  SearchConfig,
} from "./types";

export type { ApiResponse };

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export interface IApiClient {
  setApiKey(key: string): void;
  setBaseUrl(url: string): void;
  getDownloads(params?: { status?: string[]; limit?: number; offset?: number }): Promise<ApiResponse<Download[]>>;
  getDownload(id: string): Promise<Download>;
  createDownload(url: string, destination?: string, filename?: string): Promise<Download>;
  pauseDownload(id: string): Promise<void>;
  resumeDownload(id: string): Promise<void>;
  deleteDownload(id: string, deleteFiles?: boolean): Promise<void>;
  getProviders(): Promise<ApiResponse<Provider[]>>;
  configureProvider(name: string, config: Record<string, string>, enabled: boolean): Promise<void>;
  resolveUrl(url: string): Promise<any>;
  getRemotes(): Promise<ApiResponse<Remote[]>>;
  getStats(): Promise<Stats>;
  getSettings(): Promise<Record<string, string>>;
  updateSettings(settings: Record<string, string>): Promise<void>;
  listFiles(path: string): Promise<ApiResponse<FileInfo[]>>;
  mkdir(path: string): Promise<void>;
  deleteFile(path: string): Promise<void>;
  operateFile(op: 'rename' | 'copy' | 'move', src: string, dst: string): Promise<{ jobId?: string }>;
  search(q: string): Promise<ApiResponse<FileInfo[]>>;
  checkMagnet(magnet: string): Promise<MagnetInfo>;
  checkTorrent(torrentBase64: string): Promise<MagnetInfo>;
  downloadMagnet(req: MagnetDownloadRequest): Promise<Download>;
  getFileUrl(path: string): string;
  subscribeEvents(handler: (event: any) => void): () => void;
  
  // System/Admin
  getVersion(): Promise<{ version: string; aria2: string; rclone: string }>;
  restartAria2(): Promise<void>;
  restartRclone(): Promise<void>;
  restartServer(): Promise<void>;
  
  // Search Admin
  getSearchConfigs(): Promise<ApiResponse<SearchConfig[]>>;
  updateSearchConfig(remote: string, config: any): Promise<void>;
  updateSearchConfigs(configs: any): Promise<void>;
  triggerIndex(remote: string): Promise<void>;
  
  // Settings Admin
  exportSettings(): Promise<Record<string, string>>;
  importSettings(settings: Record<string, string>): Promise<void>;
  resetSettings(): Promise<void>;
  
  // File Admin
  purgeFileCache(): Promise<void>;
}

class RestApiClient implements IApiClient {
  private apiKey: string = "";
  private baseUrl: string = "/api/v1";

  setApiKey(key: string) { this.apiKey = key; }
  setBaseUrl(url: string) { this.baseUrl = url.replace(/\/+$/, ""); }

  private async request<T>(method: string, path: string, body?: any): Promise<T> {
    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (this.apiKey) headers["X-API-Key"] = this.apiKey;
    const response = await fetch(`${this.baseUrl}${path}`, { method, headers, body: body ? JSON.stringify(body) : undefined });
    if (!response.ok) throw new ApiError(response.status, await response.text() || response.statusText);
    if (response.status === 204) return {} as T;
    const text = await response.text();
    return text ? JSON.parse(text) : ({} as T);
  }

  getDownloads(params?: any) {
    const q = new URLSearchParams();
    if (params?.status) q.append("status", params.status.join(","));
    if (params?.limit) q.append("limit", params.limit.toString());
    if (params?.offset) q.append("offset", params.offset.toString());
    return this.request<ApiResponse<Download[]>>("GET", `/downloads?${q.toString()}`);
  }
  getDownload(id: string) { return this.request<Download>("GET", `/downloads/${id}`); }
  createDownload(url: string, destination?: string, filename?: string) { return this.request<Download>("POST", "/downloads", { url, destination, filename }); }
  pauseDownload(id: string) { return this.request<void>("POST", `/downloads/${id}/pause`); }
  resumeDownload(id: string) { return this.request<void>("POST", `/downloads/${id}/resume`); }
  deleteDownload(id: string, deleteFiles = false) { return this.request<void>("DELETE", `/downloads/${id}?deleteFiles=${deleteFiles}`); }
  getProviders() { return this.request<ApiResponse<Provider[]>>("GET", "/providers"); }
  configureProvider(name: string, config: any, enabled: boolean) { return this.request<void>("PUT", `/providers/${name}`, { config, enabled }); }
  resolveUrl(url: string) { return this.request<any>("POST", "/providers/resolve", { url }); }
  getRemotes() { return this.request<ApiResponse<Remote[]>>("GET", "/remotes"); }
  getStats() { return this.request<Stats>("GET", "/stats"); }
  getSettings() { return this.request<Record<string, string>>("GET", "/settings"); }
  updateSettings(settings: any) { return this.request<void>("PATCH", "/settings", settings); }
  listFiles(path: string) { return this.request<ApiResponse<FileInfo[]>>("GET", `/files/list?path=${encodeURIComponent(path)}`); }
  mkdir(path: string) { return this.request<void>("POST", "/files/mkdir", { path }); }
  deleteFile(path: string) { return this.request<void>("POST", "/files/delete", { path }); }
  operateFile(op: any, src: string, dst: string) { return this.request<{ jobId?: string }>('POST', '/files/operate', { op, src, dst }); }
  search(q: string) { return this.request<ApiResponse<FileInfo[]>>("GET", `/search?q=${encodeURIComponent(q)}`); }
  checkMagnet(magnet: string) { return this.request<MagnetInfo>("POST", "/magnets/check", { magnet }); }
  checkTorrent(torrentBase64: string) { return this.request<MagnetInfo>("POST", "/magnets/check-torrent", { torrentBase64 }); }
  downloadMagnet(req: MagnetDownloadRequest) { return this.request<Download>("POST", "/magnets/download", req); }
  getFileUrl(path: string) { return `${this.baseUrl}/files/cat?path=${encodeURIComponent(path)}`; }

  subscribeEvents(handler: (event: any) => void) {
    const evtSource = new EventSource(`${this.baseUrl}/events`);
    evtSource.onmessage = (e) => handler(JSON.parse(e.data));
    return () => evtSource.close();
  }

  getVersion() { return this.request<{ version: string; aria2: string; rclone: string }>("GET", "/system/version"); }
  restartAria2() { return this.request<void>("POST", "/system/restart/aria2"); }
  restartRclone() { return this.request<void>("POST", "/system/restart/rclone"); }
  restartServer() { return this.request<void>("POST", "/system/restart/server"); }
  getSearchConfigs() { return this.request<ApiResponse<SearchConfig[]>>('GET', '/search/config'); }
  updateSearchConfig(remote: string, config: any) { return this.request<void>('POST', `/search/config/${remote}`, config); }
  updateSearchConfigs(configs: any) { return this.request<void>('POST', '/search/config', { configs }); }
  triggerIndex(remote: string) { return this.request<void>('POST', `/search/index/${remote}`); }
  exportSettings() { return this.request<Record<string, string>>("POST", "/settings/export"); }
  importSettings(settings: Record<string, string>) { return this.request<void>("POST", "/settings/import", settings); }
  resetSettings() { return this.request<void>("POST", "/settings/reset"); }
  purgeFileCache() { return this.request<void>('POST', '/files/purge-cache'); }
}

class WailsApiClient implements IApiClient {
  private getBridge() {
     return (window as any).go?.main?.Bridge || (window as any).wails?.bindings?.main?.Bridge;
  }

  setApiKey(_key: string) {}
  setBaseUrl(_url: string) {}

  async getDownloads(params?: any) { return this.getBridge().GetDownloads(params?.status || [], params?.limit || 100, params?.offset || 0); }
  async getDownload(id: string) { return this.getBridge().GetDownload(id); }
  async createDownload(url: string, destination?: string, filename?: string) { return this.getBridge().CreateDownload(url, destination || "", filename || ""); }
  async pauseDownload(id: string) { return this.getBridge().PauseDownload(id); }
  async resumeDownload(id: string) { return this.getBridge().ResumeDownload(id); }
  async deleteDownload(id: string, deleteFiles = false) { return this.getBridge().DeleteDownload(id, deleteFiles); }
  async getProviders() { return this.getBridge().GetProviders(); }
  async configureProvider(name: string, config: any, enabled: boolean) { return this.getBridge().ConfigureProvider(name, config, enabled); }
  async resolveUrl(url: string) { return this.getBridge().ResolveUrl(url); }
  async getRemotes() { return this.getBridge().GetRemotes(); }
  async getStats() { return this.getBridge().GetStats(); }
  async getSettings() { return this.getBridge().GetSettings(); }
  async updateSettings(settings: any) { return this.getBridge().UpdateSettings(settings); }
  async listFiles(path: string) { return this.getBridge().ListFiles(path); }
  async mkdir(path: string) { return this.getBridge().Mkdir(path); }
  async deleteFile(path: string) { return this.getBridge().DeleteFile(path); }
  async operateFile(op: any, src: string, dst: string) { return this.getBridge().OperateFile(op, src, dst); }
  async search(q: string) { return this.getBridge().Search(q, 100, 0); }
  async checkMagnet(magnet: string) { return this.getBridge().CheckMagnet(magnet); }
  async checkTorrent(torrentBase64: string) { return this.getBridge().CheckTorrent(torrentBase64); }
  async downloadMagnet(req: MagnetDownloadRequest) { return this.getBridge().DownloadMagnet(req); }
  getFileUrl(path: string) { return `/api/v1/files/cat?path=${encodeURIComponent(path)}`; }

  subscribeEvents(handler: (event: any) => void) {
    const unsub = (window as any).wails?.events?.on("gravity-event", (evt: any) => {
       handler(evt.data);
    });
    return () => unsub && unsub();
  }

  async getVersion() { return { version: "0.1.0", aria2: "unknown", rclone: "unknown" }; }
  async restartAria2() {}
  async restartRclone() {}
  async restartServer() {}
  async getSearchConfigs() { return { data: [] }; }
  async updateSearchConfig(_remote: string, _config: any) {}
  async updateSearchConfigs(_configs: any) {}
  async triggerIndex(_remote: string) {}
  async exportSettings() { return {}; }
  async importSettings(_settings: Record<string, string>) {}
  async resetSettings() {}
  async purgeFileCache() {}
}

const isWails = typeof window !== 'undefined' && ((window as any).go || (window as any).wails);
export const api: IApiClient = isWails ? new WailsApiClient() : new RestApiClient();
