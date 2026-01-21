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
  seeders?: number;
  peers?: number;
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

  // Multi-file support
  isMagnet?: boolean;
  magnetHash?: string;
  magnetSource?: 'alldebrid' | 'aria2';
  files?: DownloadFile[];
  peerDetails?: Peer[];
  totalFiles?: number;
  filesComplete?: number;
}

export interface Peer {
  ip: string;
  port: string;
  downloadSpeed: number;
  uploadSpeed: number;
  isSeeder: boolean;
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

export interface DownloadFile {
  id: string;
  name: string;
  path: string;
  size: number;
  downloaded: number;
  progress: number;
  status: 'pending' | 'active' | 'complete' | 'error';
  error?: string;
}

export interface MagnetInfo {
  source: 'alldebrid' | 'aria2';
  cached: boolean;
  magnetId?: string;
  name: string;
  hash: string;
  size: number;
  files: MagnetFile[];
}

export interface MagnetFile {
  id: string;
  name: string;
  path: string;
  size: number;
  isFolder: boolean;
  children?: MagnetFile[];
  index?: number;
  link?: string;
}

export interface MagnetDownloadRequest {
  magnet: string;
  torrentBase64?: string;
  source: string;
  magnetId?: string;
  name: string;
  selectedFiles: string[];
  destination?: string;
  files: any[];
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
  speeds: {
    download: number;
    upload: number;
  };
  tasks: {
    active: number;
    uploading: number;
    waiting: number;
    paused: number;
    completed: number;
    failed: number;
  };
  usage: {
    totalDownloaded: number;
    totalUploaded: number;
    sessionDownloaded: number;
    sessionUploaded: number;
  };
  system: {
    diskFree: number;
    diskTotal: number;
    diskUsage: number;
    uptime: number;
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

export interface FileInfo {
  path: string;
  name: string;
  size: number;
  mimeType: string;
  modTime: string;
  type: 'file' | 'folder';
  isDir: boolean;
  remote?: string;
}

export interface Settings {
  download: DownloadSettings;
  upload: UploadSettings;
  network: NetworkSettings;
  torrent: TorrentSettings;
  vfs: VfsSettings;
}

export interface DownloadSettings {
  downloadDir: string;
  maxConcurrentDownloads: number;
  maxDownloadSpeed: string;
  maxUploadSpeed: string;
  maxConnectionPerServer: number;
  split: number;
  userAgent: string;
  connectTimeout: number;
  maxTries: number;
  checkCertificate: boolean;
}

export interface UploadSettings {
  defaultRemote: string;
  autoUpload: boolean;
  removeLocal: boolean;
}

export interface NetworkSettings {
  proxyEnabled: boolean;
  proxyUrl: string;
  proxyUser: string;
  proxyPassword: string;
}

export interface TorrentSettings {
  seedRatio: string;
  seedTime: number;
  listenPort: number;
  forceSave: boolean;
  enablePex: boolean;
  enableDht: boolean;
  enableLpd: boolean;
  encryption: string;
}

export interface VfsSettings {
  cacheMode: string;
  cacheMaxSize: string;
  cacheMaxAge: string;
  writeBack: string;
  readChunkSize: string;
  readChunkSizeLimit: string;
  readAhead: string;
  dirCacheTime: string;
  pollInterval: string;
  readChunkStreams: number;
}

export interface SearchConfig {
  remote: string;
  autoIndexIntervalMin: number;
  lastIndexedAt?: string;
  status: 'idle' | 'indexing' | 'error';
  errorMsg?: string;
  excludedPatterns?: string;
  includedExtensions?: string;
  minSizeBytes?: number;
}
