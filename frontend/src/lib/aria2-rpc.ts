export interface Aria2GlobalStat {
	downloadSpeed: string;
	uploadSpeed: string;
	numActive: string;
	numWaiting: string;
	numStopped: string;
	numStoppedTotal: string;
	// Augmented stats from backend
	cloudUploadSpeed?: string;
	numUploading?: string;
	totalDownloaded?: string;
	totalUploaded?: string;
	totalTasks?: string;
	completedTasks?: string;
	uploadedTasks?: string;
}

export interface Aria2Task {
	gid: string;
	status: "active" | "waiting" | "paused" | "error" | "complete" | "removed";
	totalLength: string;
	completedLength: string;
	uploadLength: string;
	bitfield: string;
	downloadSpeed: string;
	uploadSpeed: string;
	infoHash?: string;
	numSeeders?: string;
	seeder?: string;
	connections: string;
	errorCode?: string;
	errorMessage?: string;
	files: Aria2File[];
	bittorrent?: {
		announceList: string[][];
		comment?: string;
		creationDate?: number;
		mode?: "multi" | "single";
		info?: {
			name?: string;
		};
	};
	dir: string;
	rclone?: {
		status: string;
		targetRemote: string;
		jobId?: string;
		// Live upload stats
		speed?: number;
		bytes?: number;
		totalBytes?: number;
		eta?: number;
		percentage?: number;
	};
}

export interface Aria2File {
	index: string;
	path: string;
	length: string;
	completedLength: string;
	selected: string;
	uris: { uri: string; status: string }[];
}

export interface RcloneStats {
	bytes: number;
	checks: number;
	deletedDirs: number;
	deletes: number;
	elapsedTime: number;
	errors: number;
	eta: number | null;
	fatalError: boolean;
	renames: number;
	retryError: boolean;
	speed: number;
	totalBytes: number;
	totalChecks: number;
	totalTransfers: number;
	transferTime: number;
	transfers: number;
	transferring: {
		bytes: number;
		eta: number | null;
		group: string;
		name: string;
		percentage: number;
		size: number;
		speed: number;
		speedAvg: number;
	}[];
}

export interface RcloneVersion {
	arch: string;
	goVersion: string;
	isBeta: boolean;
	isGit: boolean;
	os: string;
	version: string;
}

export interface JsonRpcRequest {
	jsonrpc: "2.0";
	method: string;
	params: any[];
	id: string;
}

export interface JsonRpcResponse<T> {
	jsonrpc: "2.0";
	result?: T;
	error?: {
		code: number;
		message: string;
	};
	id: string;
}

export class Aria2Client {
	private url: string;
	private secret: string;
	private ws: WebSocket | null = null;
	private pendingRequests = new Map<
		string,
		{ resolve: (val: any) => void; reject: (err: any) => void }
	>();
	private connectionPromise: Promise<WebSocket> | null = null;

	constructor(url: string = "", secret: string = "") {
		this.url = url;
		this.secret = secret;
	}

	updateConfig(url: string, secret: string) {
		const oldUrl = this.url;
		this.url = url;
		this.secret = secret;

		// If URL changed or switching protocols, close existing connection
		if (oldUrl !== url) {
			this.closeWebSocket();
		}
	}

	private closeWebSocket() {
		if (this.ws) {
			// Clear listeners
			this.ws.onclose = null;
			this.ws.onmessage = null;
			this.ws.onerror = null;
			this.ws.onopen = null;

			this.ws.close();
			this.ws = null;
			this.connectionPromise = null;

			// Reject all pending requests
			this.pendingRequests.forEach((p) => {
				p.reject(
					new Error("WebSocket connection closed or configuration changed"),
				);
			});
			this.pendingRequests.clear();
		}
	}

	private async getWebSocket(): Promise<WebSocket> {
		if (this.ws && this.ws.readyState === WebSocket.OPEN) {
			return this.ws;
		}

		if (this.connectionPromise) {
			return this.connectionPromise;
		}

		this.connectionPromise = new Promise((resolve, reject) => {
			try {
				const ws = new WebSocket(this.url);

				ws.onopen = () => {
					this.ws = ws;
					resolve(ws);
				};

				ws.onerror = () => {
					this.connectionPromise = null;
					reject(new Error("WebSocket connection failed"));
				};

				ws.onclose = () => {
					this.ws = null;
					this.connectionPromise = null;
				};

				ws.onmessage = (event) => this.handleWsMessage(event);
			} catch (err) {
				this.connectionPromise = null;
				reject(err);
			}
		});

		return this.connectionPromise;
	}

	private handleWsMessage(event: MessageEvent) {
		try {
			const data = JSON.parse(event.data);

			// Aria2 can send notifications (no id) or responses (has id)
			// For now, we only care about responses to our requests

			const handleResponse = (res: JsonRpcResponse<any>) => {
				if (res.id && this.pendingRequests.has(res.id)) {
					const { resolve, reject } = this.pendingRequests.get(res.id)!;
					this.pendingRequests.delete(res.id);

					if (res.error) {
						reject(
							new Error(`RPC Error ${res.error.code}: ${res.error.message}`),
						);
					} else {
						resolve(res.result);
					}
				}
			};

			if (Array.isArray(data)) {
				data.forEach(handleResponse);
			} else {
				handleResponse(data);
			}
		} catch (e) {
			console.error("Failed to parse WebSocket message", e);
		}
	}

	private async request<T>(method: string, params: any[] = []): Promise<T> {
		const id = Math.random().toString(36).substring(7);
		
		// Don't inject secret for Rclone methods or if secret is empty
		const shouldInjectSecret = this.secret && !method.startsWith("rclone.") && method !== "system.multicall";
		const finalParams = shouldInjectSecret ? [`token:${this.secret}`, ...params] : params;

		const payload: JsonRpcRequest = {
			jsonrpc: "2.0",
			method,
			params: finalParams,
			id,
		};

		// WebSocket Strategy
		if (this.url.startsWith("ws://") || this.url.startsWith("wss://")) {
			try {
				const ws = await this.getWebSocket();

				return new Promise<T>((resolve, reject) => {
					this.pendingRequests.set(id, { resolve, reject });
					ws.send(JSON.stringify(payload));

					// 10s timeout
					setTimeout(() => {
						if (this.pendingRequests.has(id)) {
							this.pendingRequests.delete(id);
							reject(new Error("WebSocket request timeout"));
						}
					}, 10000);
				});
			} catch (error) {
				console.error(`Aria2 WS Error (${method}):`, error);
				throw error;
			}
		}

		// HTTP Strategy
		try {
			const response = await fetch(this.url, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
				},
				body: JSON.stringify(payload),
			});

			if (!response.ok) {
				throw new Error(`HTTP Error: ${response.status}`);
			}

			const data: JsonRpcResponse<T> = await response.json();

			if (data.error) {
				throw new Error(`RPC Error ${data.error.code}: ${data.error.message}`);
			}

			return data.result as T;
		} catch (error) {
			console.error(`Aria2 RPC Error (${method}):`, error);
			throw error;
		}
	}

	// --- Core Methods ---

	async getVersion() {
		return this.request<{ version: string; enabledFeatures: string[] }>(
			"aria2.getVersion",
		);
	}

	async getGlobalStat() {
		return this.request<Aria2GlobalStat>("aria2.getGlobalStat");
	}

	async tellActive(keys?: string[]) {
		return this.request<Aria2Task[]>("aria2.tellActive", keys ? [keys] : []);
	}

	async tellWaiting(offset: number, num: number, keys?: string[]) {
		return this.request<Aria2Task[]>("aria2.tellWaiting", [
			offset,
			num,
			keys || [],
		]);
	}

	async tellStopped(offset: number, num: number, keys?: string[]) {
		return this.request<Aria2Task[]>("aria2.tellStopped", [
			offset,
			num,
			keys || [],
		]);
	}

	async tellUploading() {
		return this.request<Aria2Task[]>("aria2.tellUploading");
	}

	async tellStatus(gid: string, keys?: string[]) {
		return this.request<Aria2Task>("aria2.tellStatus", [gid, keys || []]);
	}

	async getFiles(gid: string) {
		return this.request<Aria2File[]>("aria2.getFiles", [gid]);
	}

	async getPeers(gid: string) {
		return this.request<any[]>("aria2.getPeers", [gid]);
	}

	async getServers(gid: string) {
		return this.request<any[]>("aria2.getServers", [gid]);
	}

	async changePosition(
		gid: string,
		pos: number,
		how: "pos-set" | "pos-cur" | "pos-end",
	) {
		return this.request<number>("aria2.changePosition", [gid, pos, how]);
	}

	async getUris(gid: string) {
		return this.request<any[]>("aria2.getUris", [gid]);
	}

	// --- Actions ---

	async addUri(uris: string[], options: Record<string, string> = {}) {
		return this.request<string>("aria2.addUri", [uris, options]);
	}

	async addTorrent(
		torrentBase64: string,
		uris: string[] = [],
		options: Record<string, string> = {},
	) {
		return this.request<string>("aria2.addTorrent", [
			torrentBase64,
			uris,
			options,
		]);
	}

	async addMetalink(
		metalinkBase64: string,
		options: Record<string, string> = {},
	) {
		return this.request<string>("aria2.addMetalink", [metalinkBase64, options]);
	}

	async remove(gid: string) {
		return this.request<string>("aria2.remove", [gid]);
	}

	async forceRemove(gid: string) {
		return this.request<string>("aria2.forceRemove", [gid]);
	}

	async pause(gid: string) {
		return this.request<string>("aria2.pause", [gid]);
	}

	async pauseAll() {
		return this.request<string>("aria2.pauseAll");
	}

	async forcePause(gid: string) {
		return this.request<string>("aria2.forcePause", [gid]);
	}

	async forcePauseAll() {
		return this.request<string>("aria2.forcePauseAll");
	}

	async unpause(gid: string) {
		return this.request<string>("aria2.unpause", [gid]);
	}

	async unpauseAll() {
		return this.request<string>("aria2.unpauseAll");
	}

	// --- Options ---

	async getOption(gid: string) {
		return this.request<Record<string, string>>("aria2.getOption", [gid]);
	}

	async changeOption(gid: string, options: Record<string, string>) {
		return this.request<string>("aria2.changeOption", [gid, options]);
	}

	async getGlobalOption() {
		return this.request<Record<string, string>>("aria2.getGlobalOption");
	}

	async changeGlobalOption(options: Record<string, string>) {
		return this.request<string>("aria2.changeGlobalOption", [options]);
	}

	async removeDownloadResult(gid: string) {
		return this.request<string>("aria2.removeDownloadResult", [gid]);
	}

	async purgeDownloadResult() {
		return this.request<string>("aria2.purgeDownloadResult");
	}

	async saveSession() {
		return this.request<string>("aria2.saveSession");
	}

	async retryTask(gid: string) {
		return this.request<string>("aria2.retryTask", [gid]);
	}

	// --- System ---

	async multicall(methods: { methodName: string; params: any[] }[]) {
		// Manually construct multicall payload as it's structurally different
		return this.request<any[]>("system.multicall", [
			methods.map((m) => ({
				methodName: m.methodName,
				params: this.secret ? [`token:${this.secret}`, ...m.params] : m.params,
			})),
		]);
	}

	// --- Rclone Methods ---

	async rcloneListRemotes() {
		// Backend expects empty params object or just empty params list which defaults to empty object
		return this.request<{ remotes: string[] }>("rclone.listRemotes", [{}]).then(
			(r) => r.remotes,
		);
	}

	async rcloneGetStats() {
		return this.request<RcloneStats>("rclone.getStats", [{}]);
	}

	async rcloneGetVersion() {
		return this.request<RcloneVersion>("rclone.getVersion", [{}]);
	}

	async rcloneCreateRemote(name: string, type: string, parameters: Record<string, string>) {
		return this.request<any>("rclone.createRemote", [{ name, type, parameters }]);
	}

	async rcloneDeleteRemote(name: string) {
		return this.request<any>("rclone.deleteRemote", [{ name }]);
	}
}

export const aria2 = new Aria2Client();
