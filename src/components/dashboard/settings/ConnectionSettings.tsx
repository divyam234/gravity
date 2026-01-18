import {
	Button,
	Card,
	Chip,
	FieldError,
	Form,
	Input,
	Label,
	ListBox,
	Select,
	TextField,
} from "@heroui/react";
import type React from "react";
import { useId, useState } from "react";
import IconCheck from "~icons/gravity-ui/check";
import IconChevronDown from "~icons/gravity-ui/chevron-down";
import IconCircleInfo from "~icons/gravity-ui/circle-info";
import IconFloppyDisk from "~icons/gravity-ui/floppy-disk";
import IconPencil from "~icons/gravity-ui/pencil";
import IconPlus from "~icons/gravity-ui/plus";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import { useAria2Actions, useAria2Version } from "../../../hooks/useAria2";
import { cn } from "../../../lib/utils";
import {
	type ServerConfig,
	useSettingsStore,
} from "../../../store/useSettingsStore";

export const ConnectionSettings: React.FC = () => {
	const baseId = useId();
	const {
		servers,
		activeServerId,
		addServer,
		updateServer,
		removeServer,
		setActiveServer,
	} = useSettingsStore();
	const { data: version } = useAria2Version();
	const { saveSession } = useAria2Actions();

	// Local state for editing/adding
	const [editingId, setEditingId] = useState<string | null>(null);
	const [isAdding, setIsAdding] = useState(false);
	const [isTesting, setIsTesting] = useState(false);
	const [testResult, setTestResult] = useState<{
		success: boolean;
		message: string;
	} | null>(null);

	// Form state
	const [formData, setFormData] = useState<Omit<ServerConfig, "id">>({
		name: "",
		rpcUrl: "http://localhost:6800/jsonrpc",
		rpcSecret: "",
	});

	// Check if current connection is HTTPS (to disable non-secure options if needed)
	const isSecureContext = window.location.protocol === "https:";

	const resetForm = () => {
		setFormData({
			name: "",
			rpcUrl: "http://localhost:6800/jsonrpc",
			rpcSecret: "",
		});
		setEditingId(null);
		setIsAdding(false);
	};

	const startAdd = () => {
		resetForm();
		setIsAdding(true);
	};

	const startEdit = (server: ServerConfig) => {
		setFormData({
			name: server.name,
			rpcUrl: server.rpcUrl,
			rpcSecret: server.rpcSecret,
		});
		setEditingId(server.id);
		setIsAdding(false);
	};

	const handleProtocolChange = (key: string) => {
		let cleanUrl = formData.rpcUrl.replace(/^(https?|wss?):\/\//, "");
		if (!cleanUrl) cleanUrl = "localhost:6800/jsonrpc";

		const prefix =
			key === "websocket"
				? "ws://"
				: key === "websocket-secure"
					? "wss://"
					: key === "https"
						? "https://"
						: "http://";

		setFormData({ ...formData, rpcUrl: prefix + cleanUrl });
	};

	const handleSave = (e: React.FormEvent) => {
		e.preventDefault();
		if (editingId) {
			updateServer(editingId, formData);
		} else {
			addServer(formData);
		}
		resetForm();
	};

	const handleTestConnection = async () => {
		setIsTesting(true);
		setTestResult(null);
		try {
			// Create a temporary client to test
			const testClient = new (
				await import("../../../lib/aria2-rpc")
			).Aria2Client(formData.rpcUrl, formData.rpcSecret);
			const ver = await testClient.getVersion();
			setTestResult({
				success: true,
				message: `Connected! Aria2 version ${ver.version}`,
			});
		} catch (err: any) {
			setTestResult({
				success: false,
				message: `Failed: ${err.message || "Unknown error"}`,
			});
		} finally {
			setIsTesting(false);
		}
	};

	const handleDelete = (id: string) => {
		if (confirm("Are you sure you want to delete this server configuration?")) {
			removeServer(id);
		}
	};

	const validateUrl = (val: string) => {
		if (!val) return "URL is required";
		try {
			const url = new URL(val);
			if (
				url.protocol !== "http:" &&
				url.protocol !== "https:" &&
				url.protocol !== "ws:" &&
				url.protocol !== "wss:"
			) {
				return "Must be http/https or ws/wss";
			}
		} catch {
			return "Invalid URL format";
		}
		return true;
	};

	const isFormVisible = isAdding || editingId !== null;

	const getProtocol = (url: string) => {
		if (url.startsWith("wss://")) return "websocket-secure";
		if (url.startsWith("ws://")) return "websocket";
		if (url.startsWith("https://")) return "https";
		return "http";
	};

	return (
		<div className="space-y-8">
			{/* Server List Section */}
			<div className="space-y-6">
				<div className="flex items-center justify-between border-b border-border pb-2">
					<div>
						<h3 className="text-lg font-bold">RPC Servers</h3>
						<p className="text-sm text-muted">
							Manage your Aria2 server connections.
						</p>
					</div>
					{!isFormVisible && (
						<Button
							size="sm"
							variant="primary"
							onPress={startAdd}
							className="font-bold"
						>
							<IconPlus className="w-4 h-4 mr-1" />
							Add Server
						</Button>
					)}
				</div>

				{/* Edit/Add Form */}
				{isFormVisible && (
					<Card className="p-6 border border-accent/20 bg-accent/5">
						<h4 className="text-sm font-bold uppercase tracking-wider mb-4 flex items-center gap-2">
							{isAdding ? (
								<IconPlus className="w-4 h-4" />
							) : (
								<IconPencil className="w-4 h-4" />
							)}
							{isAdding ? "Add New Server" : "Edit Server"}
						</h4>

						<Form className="space-y-4" onSubmit={handleSave}>
							<TextField
								isRequired
								value={formData.name}
								onChange={(val) => setFormData({ ...formData, name: val })}
							>
								<Label className="text-sm font-bold">Name</Label>
								<Input className="bg-background" placeholder="My Home Server" />
								<FieldError />
							</TextField>

							<div className="flex flex-col gap-2">
								<Label className="text-sm font-bold">Connection Protocol</Label>
								<Select
									selectedKey={`${baseId}-${getProtocol(formData.rpcUrl)}`}
									onSelectionChange={(key) =>
										handleProtocolChange(String(key).replace(`${baseId}-`, ""))
									}
									className="max-w-full"
								>
									<Select.Trigger className="bg-background">
										<Select.Value />
										<Select.Indicator>
											<IconChevronDown />
										</Select.Indicator>
									</Select.Trigger>
									<Select.Popover className="p-1 bg-background border border-border rounded-xl shadow-xl min-w-[300px]">
										<ListBox>
											<ListBox.Item
												id={`${baseId}-http`}
												textValue="HTTP (Standard)"
												className={cn(
													"px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 cursor-pointer outline-none flex flex-col",
													isSecureContext && "opacity-50 pointer-events-none",
												)}
											>
												<div className="flex flex-col gap-0.5">
													<span className="text-sm font-bold">
														HTTP (Standard)
													</span>
													<span className="text-xs text-muted">
														Standard unencrypted connection (ws://)
														{isSecureContext && " (Disabled on HTTPS)"}
													</span>
												</div>
											</ListBox.Item>
											<ListBox.Item
												id={`${baseId}-https`}
												textValue="HTTPS (Secure)"
												className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 cursor-pointer outline-none flex flex-col"
											>
												<div className="flex flex-col gap-0.5">
													<span className="text-sm font-bold">
														HTTPS (Secure)
													</span>
													<span className="text-xs text-muted">
														Encrypted HTTP connection
													</span>
												</div>
											</ListBox.Item>
											<ListBox.Item
												id={`${baseId}-websocket`}
												textValue="WebSocket (Real-time)"
												className={cn(
													"px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 cursor-pointer outline-none flex flex-col",
													isSecureContext && "opacity-50 pointer-events-none",
												)}
											>
												<div className="flex flex-col gap-0.5">
													<span className="text-sm font-bold">
														WebSocket (Real-time)
													</span>
													<span className="text-xs text-muted">
														Live updates via ws://
														{isSecureContext && " (Disabled on HTTPS)"}
													</span>
												</div>
											</ListBox.Item>
											<ListBox.Item
												id={`${baseId}-websocket-secure`}
												textValue="WebSocket Secure (WSS)"
												className="px-3 py-2 rounded-lg data-[hover=true]:bg-default/15 cursor-pointer outline-none flex flex-col"
											>
												<div className="flex flex-col gap-0.5">
													<span className="text-sm font-bold">
														WebSocket Secure (WSS)
													</span>
													<span className="text-xs text-muted">
														Encrypted live updates via wss://
													</span>
												</div>
											</ListBox.Item>
										</ListBox>
									</Select.Popover>
								</Select>
							</div>

							<TextField
								isRequired
								value={formData.rpcUrl}
								onChange={(val) => setFormData({ ...formData, rpcUrl: val })}
								validate={validateUrl}
							>
								<Label className="text-sm font-bold">RPC URL</Label>
								<Input
									className="bg-background"
									placeholder="http://localhost:6800/jsonrpc"
								/>
								<FieldError />
							</TextField>

							<TextField
								value={formData.rpcSecret}
								onChange={(val) => setFormData({ ...formData, rpcSecret: val })}
							>
								<Label className="text-sm font-bold">RPC Secret</Label>
								<Input
									className="bg-background"
									type="password"
									placeholder="Optional"
								/>
							</TextField>

							{testResult && (
								<div
									className={cn(
										"text-xs font-bold p-3 rounded-xl border animate-in fade-in slide-in-from-top-1",
										testResult.success
											? "bg-success/10 border-success/30 text-success"
											: "bg-danger/10 border-danger/30 text-danger",
									)}
								>
									{testResult.message}
								</div>
							)}

							<div className="flex justify-between items-center pt-2">
								<Button
									size="sm"
									variant="secondary"
									onPress={handleTestConnection}
									isDisabled={isTesting || !formData.rpcUrl}
									className="font-bold"
								>
									{isTesting ? "Testing..." : "Test Connection"}
								</Button>
								<div className="flex gap-2">
									<Button size="sm" variant="secondary" onPress={resetForm}>
										Cancel
									</Button>
									<Button size="sm" variant="primary" type="submit">
										Save Server
									</Button>
								</div>
							</div>
						</Form>
					</Card>
				)}

				{/* List of Servers */}
				<div className="grid gap-3">
					{servers.map((server) => (
						<div
							key={server.id}
							className={cn(
								"group relative p-4 rounded-2xl border transition-all duration-200",
								activeServerId === server.id
									? "bg-accent/10 border-accent shadow-sm"
									: "bg-muted-background border-border hover:border-accent/50",
							)}
						>
							<div className="flex items-center justify-between">
								<div className="flex flex-col gap-1">
									<div className="flex items-center gap-2">
										<span className="font-bold text-base">{server.name}</span>
										{activeServerId === server.id && (
											<Chip
												size="sm"
												color="success"
												variant="soft"
												className="h-5 px-1"
											>
												<span className="flex items-center gap-1 text-[10px] font-bold uppercase">
													<IconCheck className="w-3 h-3" /> Active
												</span>
											</Chip>
										)}
									</div>
									<span className="text-xs font-mono text-muted truncate max-w-[300px]">
										{server.rpcUrl}
									</span>
								</div>

								<div className="flex items-center gap-2">
									{activeServerId !== server.id && (
										<Button
											size="sm"
											variant="ghost"
											onPress={() => setActiveServer(server.id)}
											className="font-bold text-xs"
										>
											Connect
										</Button>
									)}

									<div className="flex items-center gap-1 border-l border-border/50 pl-2 ml-2">
										<Button
											isIconOnly
											size="sm"
											variant="ghost"
											onPress={() => startEdit(server)}
										>
											<IconPencil className="w-4 h-4 text-muted-foreground" />
										</Button>
										<Button
											isIconOnly
											size="sm"
											variant="ghost"
											onPress={() => handleDelete(server.id)}
											isDisabled={servers.length <= 1} // Prevent deleting last server
										>
											<IconTrashBin className="w-4 h-4 text-danger" />
										</Button>
									</div>
								</div>
							</div>
						</div>
					))}

					{servers.length === 0 && (
						<div className="text-center p-8 border border-dashed border-border rounded-2xl text-muted">
							No servers configured. Add one to get started.
						</div>
					)}
				</div>
			</div>

			<div className="space-y-4 pt-4 border-t border-border">
				<h4 className="text-sm font-bold uppercase tracking-wider text-foreground/80">
					Manual Operations
				</h4>
				<div className="bg-muted-background p-4 rounded-2xl border border-border flex items-center justify-between">
					<div className="flex flex-col gap-0.5">
						<span className="text-sm font-bold">Save Current Session</span>
						<p className="text-xs text-muted">
							Force save the current download list to the session file.
						</p>
					</div>
					<Button
						size="sm"
						variant="secondary"
						className="font-bold"
						onPress={() => saveSession.mutate()}
						isDisabled={saveSession.isPending}
					>
						<IconFloppyDisk className="w-4 h-4 mr-2" />
						Save Now
					</Button>
				</div>
			</div>

			{version && (
				<div className="space-y-4 pt-4 border-t border-border">
					<div className="flex items-center gap-2">
						<IconCircleInfo className="w-4 h-4 text-accent" />
						<h4 className="text-sm font-bold uppercase tracking-wider text-foreground/80">
							Aria2 Instance Information
						</h4>
					</div>

					<div className="bg-muted-background p-5 rounded-3xl border border-border space-y-5 shadow-sm">
						<div className="flex justify-between items-center">
							<span className="text-sm font-bold">Version</span>
							<Chip
								size="sm"
								variant="soft"
								color="accent"
								className="font-mono font-bold"
							>
								v{version.version}
							</Chip>
						</div>

						<div className="space-y-3">
							<span className="text-xs text-muted uppercase font-black tracking-widest">
								Enabled Features
							</span>
							<div className="flex flex-wrap gap-2">
								{version.enabledFeatures.map((feature) => (
									<Chip
										key={feature}
										size="sm"
										variant="soft"
										className="text-[10px] uppercase font-bold bg-default/10"
									>
										{feature}
									</Chip>
								))}
							</div>
						</div>
					</div>
				</div>
			)}
		</div>
	);
};
