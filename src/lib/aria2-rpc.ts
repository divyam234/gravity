export interface Aria2GlobalStat {
	downloadSpeed: string;
	uploadSpeed: string;
	numActive: string;
	numWaiting: string;
	numStopped: string;
	numStoppedTotal: string;
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
}

export interface Aria2File {
	index: string;
	path: string;
	length: string;
	completedLength: string;
	selected: string;
	uris: { uri: string; status: string }[];
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

	constructor(
		url: string = "http://localhost:6800/jsonrpc",
		secret: string = "",
	) {
		this.url = url;
		this.secret = secret;
	}

	updateConfig(url: string, secret: string) {
		this.url = url;
		this.secret = secret;
	}

	private async request<T>(method: string, params: any[] = []): Promise<T> {
		const id = Math.random().toString(36).substring(7);
		const payload: JsonRpcRequest = {
			jsonrpc: "2.0",
			method,
			params: this.secret ? [`token:${this.secret}`, ...params] : params,
			id,
		};

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

	async purgeDownloadResult() {
		return this.request<string>("aria2.purgeDownloadResult");
	}

	async saveSession() {
		return this.request<string>("aria2.saveSession");
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
}

export const aria2 = new Aria2Client();
