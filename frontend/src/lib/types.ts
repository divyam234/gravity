export interface Download {
  id: string;
  url: string;
  resolvedUrl?: string;
  provider?: string;
  status: 'active' | 'waiting' | 'paused' | 'uploading' | 'complete' | 'error';
  error?: string;
  filename?: string;
  size: number;
  downloaded: number;
  speed: number;
  eta: number;
  destination?: string;
  uploadStatus?: string;
  uploadProgress: number;
  uploadSpeed: number;
  category?: string;
  tags?: string[];
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  updatedAt: string;
}

export interface Provider {
  name: string;
  displayName: string;
  type: 'direct' | 'debrid' | 'filehost';
  enabled: boolean;
  priority: number;
  config?: Record<string, string>;
  cachedHosts?: string[];
  cachedAccount?: AccountInfo;
  updatedAt: string;
}

export interface AccountInfo {
  username?: string;
  isPremium: boolean;
  expiresAt?: string;
}

export interface Remote {
  name: string;
  type: string;
  connected: boolean;
}

export interface Stats {
  active: {
    downloads: number;
    downloadSpeed: number;
    uploads: number;
    uploadSpeed: number;
  };
  queue: {
    pending: number;
    paused: number;
  };
  totals: {
    totalDownloaded: number;
    totalUploaded: number;
    tasksFinished: number;
    tasksFailed: number;
  };
}

export interface ApiResponse<T> {
  data: T;
  meta?: {
    total: number;
    limit: number;
    offset: number;
  };
}
