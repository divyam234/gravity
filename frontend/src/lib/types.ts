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
  downloadDir?: string;
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
  metadataFetching?: boolean;

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

export interface TaskOptions {
  maxDownloadSpeed?: number;
  connections?: number;
  split?: number;
  proxyUrl?: string;
  uploadRemote?: string;
  headers?: Record<string, string>;
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
  downloadDir?: string;
  destination?: string;
  files: any[];
  options?: TaskOptions;
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
  search: SearchSettings;
  advanced: AdvancedSettings;
  automation: AutomationSettings;
  updatedAt: string;
}

export interface SearchSettings {
  configs: SearchConfig[];
}

export interface AdvancedSettings {
  logLevel: "debug" | "info" | "warn" | "error";
  debugMode: boolean;
  saveInterval: number;
}

export interface AutomationSettings {
  scheduleEnabled: boolean;
  rules: ScheduleRule[];
  onCompleteAction: "none" | "shutdown" | "sleep" | "run_script";
  scriptPath: string;
  categories: Category[];
}

export interface ScheduleRule {
  id: string;
  enabled: boolean;
  label: string;
  days: number[];
  startTime: string;
  endTime: string;
  downloadLimit: string;
  uploadLimit: string;
}

export interface Category {
  id: string;
  name: string;
  path: string;
  extensions: string[];
  icon: string;
  isDefault: boolean;
}

export interface DownloadSettings {
  downloadDir: string;
  maxConcurrentDownloads: number;
  preferredEngine: "aria2" | "native";
  preferredMagnetEngine: "aria2" | "native";
  maxDownloadSpeed: string;
  maxUploadSpeed: string;
  maxConnectionPerServer: number;
  split: number;
  userAgent: string;
  connectTimeout: number;
  maxTries: number;
  checkCertificate: boolean;
  autoResume: boolean;
  preAllocateSpace: boolean;
  diskCache: string;
  minSplitSize: string;
  lowestSpeedLimit: string;
}

export interface UploadSettings {
  defaultRemote: string;
  autoUpload: boolean;
  removeLocal: boolean;
  concurrentUploads: number;
  uploadBandwidth: string;
  maxRetryAttempts: number;
  chunkSize: string;
}

export interface ProxyConfig {
  enabled: boolean;
  url: string;
  user?: string;
  password?: string;
}

export interface NetworkSettings {
  proxyMode: "global" | "granular";
  globalProxy: ProxyConfig;
  magnetProxy: ProxyConfig;
  downloadProxy: ProxyConfig;
  uploadProxy: ProxyConfig;
  dnsOverHttps: string;
  interfaceBinding: string;
  tcpPortRange: string;
}

export interface TorrentSettings {
  seedRatio: string;
  seedTime: number;
  listenPort: number;
  forceSave: boolean;
  enablePex: boolean;
  enableDht: boolean;
  enableLpd: boolean;
  encryption: "forced" | "enabled" | "disabled";
  maxPeers: number;
}

export interface VfsSettings {
  cacheMode: "off" | "minimal" | "writes" | "full";
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
