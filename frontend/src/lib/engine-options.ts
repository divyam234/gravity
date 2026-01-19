export interface EngineOptionDefinition {
    type: 'string' | 'integer' | 'float' | 'text' | 'boolean' | 'option';
    since?: string;
    suffix?: string;
    readonly?: boolean;
    defaultValue?: string;
    required?: boolean;
    separator?: string;
    overrideMode?: 'override' | 'append';
    submitFormat?: 'string' | 'array';
    showCount?: boolean;
    trimCount?: boolean;
    options?: string[];
    min?: number;
    max?: number;
    pattern?: string;
    description?: string;
}

export const engineAllOptions: Record<string, EngineOptionDefinition> = {
  "downloadDir": {
    "type": "string",
    "required": true,
    "description": "The directory to store the downloaded file."
  },
  "maxConcurrentDownloads": {
    "type": "integer",
    "defaultValue": "5",
    "required": true,
    "min": 1,
    "description": "Set the maximum number of parallel downloads."
  },
  "globalDownloadSpeedLimit": {
    "type": "string",
    "suffix": "Bytes",
    "defaultValue": "0",
    "required": true,
    "pattern": "^(0|[1-9]\d*(K|k|M|m)?)$",
    "description": "Set max overall download speed in bytes/sec. 0 means unrestricted. You can append K or M."
  },
  "globalUploadSpeedLimit": {
    "type": "string",
    "suffix": "Bytes",
    "defaultValue": "0",
    "required": true,
    "pattern": "^(0|[1-9]\d*(K|k|M|m)?)$",
    "description": "Set max overall upload speed in bytes/sec. 0 means unrestricted."
  },
  "maxConnectionsPerServer": {
    "type": "integer",
    "defaultValue": "1",
    "required": true,
    "min": 1,
    "max": 16,
    "description": "The maximum number of connections to one server for each download."
  },
  "concurrency": {
    "type": "integer",
    "defaultValue": "5",
    "required": true,
    "min": 1,
    "description": "Download a file using N connections (split)."
  },
  "userAgent": {
    "type": "string",
    "defaultValue": "gravity/1.0",
    "description": "Set user agent for HTTP(S) downloads."
  },
  "proxyUrl": {
    "type": "string",
    "description": "Use a proxy server for all protocols (e.g. http://user:pass@host:port)."
  },
  "proxyUser": {
    "type": "string",
    "description": "Set user for proxy."
  },
  "proxyPassword": {
    "type": "string",
    "description": "Set password for proxy."
  },
  "seedRatio": {
    "type": "float",
    "defaultValue": "1.0",
    "required": true,
    "min": 0,
    "description": "Specify share ratio. Seed completed torrents until share ratio reaches RATIO."
  },
  "seedTimeLimit": {
    "type": "float",
    "suffix": "Minutes",
    "required": true,
    "min": 0,
    "description": "Specify seeding time limit in minutes."
  },
  "connectTimeout": {
    "type": "integer",
    "suffix": "Seconds",
    "defaultValue": "60",
    "required": true,
    "min": 1,
    "max": 600,
    "description": "Set the connect timeout in seconds."
  },
  "maxRetries": {
    "type": "integer",
    "defaultValue": "5",
    "required": true,
    "min": 0,
    "description": "Set number of retries. 0 means unlimited."
  },
  "checkIntegrity": {
    "type": "boolean",
    "defaultValue": "false",
    "required": true,
    "description": "Check file integrity by validating piece hashes."
  },
  "continueDownloads": {
    "type": "boolean",
    "required": true,
    "description": "Continue downloading a partially downloaded file."
  },
  "checkCertificate": {
    "type": "boolean",
    "defaultValue": "true",
    "description": "Verify the peer using certificates."
  }
};

export const engineGlobalAvailableOptions: Record<string, string[]> = {
  "downloads": [
    "downloadDir",
    "maxConcurrentDownloads",
    "globalDownloadSpeedLimit",
    "globalUploadSpeedLimit",
    "continueDownloads",
    "checkIntegrity"
  ],
  "network": [
    "maxConnectionsPerServer",
    "concurrency",
    "connectTimeout",
    "maxRetries",
    "userAgent",
    "proxyUrl",
    "proxyUser",
    "proxyPassword",
    "checkCertificate"
  ],
  "bittorrent": [
    "seedRatio",
    "seedTimeLimit"
  ]
};
