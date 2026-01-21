import { Button, Card, Input, Label, Modal, ScrollShadow, Spinner } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconPlus from "~icons/gravity-ui/plus";
import IconTrashBin from "~icons/gravity-ui/trash-bin";
import IconFolder from "~icons/gravity-ui/folder";
import IconCheck from "~icons/gravity-ui/check";
import IconCloud from "~icons/gravity-ui/cloud";
import IconCloudArrowUpIn from "~icons/gravity-ui/cloud-arrow-up-in";
import { useRemotes, useRemoteActions } from "../hooks/useRemotes";
import { useSettingsStore } from "../store/useSettingsStore";
import { cn } from "../lib/utils";

export const Route = createFileRoute("/settings/cloud")({
	component: CloudSettingsPage,
});

// Provider definitions for the wizard
const CLOUD_PROVIDERS = [
	{ id: "drive", name: "Google Drive", icon: "🔵", color: "bg-blue-500/10 text-blue-500" },
	{ id: "s3", name: "Amazon S3", icon: "📦", color: "bg-orange-500/10 text-orange-500" },
	{ id: "dropbox", name: "Dropbox", icon: "🟦", color: "bg-blue-600/10 text-blue-600" },
	{ id: "onedrive", name: "OneDrive", icon: "🟢", color: "bg-green-500/10 text-green-500" },
	{ id: "sftp", name: "SFTP / FTP", icon: "📁", color: "bg-default/10 text-foreground" },
	{ id: "local", name: "Local Path", icon: "💾", color: "bg-default/10 text-foreground" },
];

interface WizardState {
	step: "select" | "configure" | "success";
	provider: typeof CLOUD_PROVIDERS[number] | null;
	name: string;
	config: Record<string, string>;
}

function CloudSettingsPage() {
	const navigate = useNavigate();
	const { defaultRemote, setDefaultRemote } = useSettingsStore();
	const { data: remotes = [], isLoading } = useRemotes();
	const { deleteRemote, createRemote } = useRemoteActions();

	const [isWizardOpen, setIsWizardOpen] = useState(false);
	const [wizard, setWizard] = useState<WizardState>({
		step: "select",
		provider: null,
		name: "",
		config: {},
	});

	const openWizard = () => {
		setWizard({ step: "select", provider: null, name: "", config: {} });
		setIsWizardOpen(true);
	};

	const closeWizard = () => {
		setIsWizardOpen(false);
	};

	const selectProvider = (provider: typeof CLOUD_PROVIDERS[number]) => {
		setWizard({ ...wizard, step: "configure", provider, name: "", config: {} });
	};

	const handleCreate = async () => {
		if (!wizard.provider || !wizard.name) return;

		try {
			await createRemote.mutateAsync({
				name: wizard.name,
				type: wizard.provider.id,
				parameters: wizard.config,
			});
			setWizard({ ...wizard, step: "success" });
		} catch (err) {
			// Error handled by mutation
		}
	};

	const handleSetDefault = (remoteName: string) => {
		setDefaultRemote(`${remoteName}:`);
	};

	const handleDelete = (remoteName: string) => {
		if (confirm(`Delete remote "${remoteName}"? This cannot be undone.`)) {
			deleteRemote.mutate(remoteName);
			if (defaultRemote.startsWith(`${remoteName}:`)) {
				setDefaultRemote("");
			}
		}
	};

	const renderConfigFields = () => {
		if (!wizard.provider) return null;

		switch (wizard.provider.id) {
			case "drive":
				return (
					<div className="space-y-4">
						<p className="text-sm text-muted">
							Enter your Google Drive configuration token or JSON from rclone.
						</p>
						<Input
							value={wizard.config.token || ""}
							onChange={(e) => setWizard({ ...wizard, config: { ...wizard.config, token: e.target.value } })}
							placeholder="Paste rclone config/token here..."
							className="h-12 bg-default/10 rounded-2xl border-none font-mono text-xs"
						/>
					</div>
				);

			case "s3":
				return (
					<div className="space-y-4">
						<div>
							<Label className="text-sm font-bold mb-2 block">Access Key ID</Label>
							<Input
								value={wizard.config.access_key_id || ""}
								onChange={(e) => setWizard({ ...wizard, config: { ...wizard.config, access_key_id: e.target.value } })}
								placeholder="AKIAIOSFODNN7EXAMPLE"
								className="h-12 bg-default/10 rounded-2xl border-none"
							/>
						</div>
						<div>
							<Label className="text-sm font-bold mb-2 block">Secret Access Key</Label>
							<Input
								type="password"
								value={wizard.config.secret_access_key || ""}
								onChange={(e) => setWizard({ ...wizard, config: { ...wizard.config, secret_access_key: e.target.value } })}
								placeholder="Your secret key"
								className="h-12 bg-default/10 rounded-2xl border-none"
							/>
						</div>
						<div>
							<Label className="text-sm font-bold mb-2 block">Region</Label>
							<Input
								value={wizard.config.region || ""}
								onChange={(e) => setWizard({ ...wizard, config: { ...wizard.config, region: e.target.value } })}
								placeholder="us-east-1"
								className="h-12 bg-default/10 rounded-2xl border-none"
							/>
						</div>
					</div>
				);

			case "sftp":
				return (
					<div className="space-y-4">
						<div>
							<Label className="text-sm font-bold mb-2 block">Host</Label>
							<Input
								value={wizard.config.host || ""}
								onChange={(e) => setWizard({ ...wizard, config: { ...wizard.config, host: e.target.value } })}
								placeholder="sftp.example.com"
								className="h-12 bg-default/10 rounded-2xl border-none"
							/>
						</div>
						<div>
							<Label className="text-sm font-bold mb-2 block">Username</Label>
							<Input
								value={wizard.config.user || ""}
								onChange={(e) => setWizard({ ...wizard, config: { ...wizard.config, user: e.target.value } })}
								placeholder="username"
								className="h-12 bg-default/10 rounded-2xl border-none"
							/>
						</div>
						<div>
							<Label className="text-sm font-bold mb-2 block">Password</Label>
							<Input
								type="password"
								value={wizard.config.pass || ""}
								onChange={(e) => setWizard({ ...wizard, config: { ...wizard.config, pass: e.target.value } })}
								placeholder="Password"
								className="h-12 bg-default/10 rounded-2xl border-none"
							/>
						</div>
					</div>
				);

			case "local":
				return (
					<div className="space-y-4">
						<div>
							<Label className="text-sm font-bold mb-2 block">Local Path</Label>
							<Input
								value={wizard.config.path || ""}
								onChange={(e) => setWizard({ ...wizard, config: { ...wizard.config, path: e.target.value } })}
								placeholder="/mnt/backup"
								className="h-12 bg-default/10 rounded-2xl border-none"
							/>
						</div>
						<p className="text-xs text-muted">
							This path must exist on the server where Gravity is running.
						</p>
					</div>
				);

			default:
				return (
					<div className="space-y-4">
						<p className="text-sm text-muted">
							Configuration for {wizard.provider.name} requires manual setup.
						</p>
						<div>
							<Label className="text-sm font-bold mb-2 block">Configuration (JSON)</Label>
							<Input
								value={wizard.config.raw || "{}"}
								onChange={(e) => setWizard({ ...wizard, config: { raw: e.target.value } })}
								placeholder="{}"
								className="h-12 bg-default/10 rounded-2xl border-none font-mono"
							/>
						</div>
					</div>
				);
		}
	};

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex items-center gap-4 px-2 shrink-0">
				<Button
					variant="ghost"
					isIconOnly
					onPress={() => navigate({ to: "/settings" })}
				>
					<IconChevronLeft className="w-5 h-5" />
				</Button>
				<div>
					<h2 className="text-2xl font-bold tracking-tight">Cloud Storage</h2>
					<p className="text-xs text-muted">Manage remotes & upload destinations</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{/* Default Destination */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Default Upload Destination</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border">
								<div className="space-y-3">
									<div className="relative">
										<IconCloudArrowUpIn className="absolute left-4 top-1/2 -translate-y-1/2 w-4 h-4 text-muted" />
										<Input
											value={defaultRemote}
											onChange={(e) => setDefaultRemote(e.target.value)}
											placeholder="gdrive:/downloads"
											className="pl-11 h-12 bg-default/10 rounded-2xl border-none"
											fullWidth
										/>
									</div>
									<p className="text-xs text-muted">
										Files are automatically uploaded here after download completes. Leave empty to keep files local.
									</p>
								</div>
							</Card>
						</section>

						{/* Remotes */}
						<section>
							<div className="flex items-center justify-between mb-6">
								<div className="flex items-center gap-3">
									<div className="w-1.5 h-6 bg-accent rounded-full" />
									<h3 className="text-lg font-bold">Your Remotes</h3>
								</div>
								<Button
									size="sm"
									variant="primary"
									onPress={openWizard}
									className="rounded-xl font-bold"
								>
									<IconPlus className="w-4 h-4 mr-1" />
									Add Remote
								</Button>
							</div>

							{isLoading ? (
								<div className="flex justify-center py-12">
									<Spinner size="md" />
								</div>
							) : remotes.length === 0 ? (
								<Card className="p-8 bg-background/50 border-border border-dashed">
									<div className="flex flex-col items-center text-center">
										<div className="w-16 h-16 bg-default/10 rounded-full flex items-center justify-center mb-4">
											<IconCloud className="w-8 h-8 text-muted" />
										</div>
										<h4 className="font-bold text-lg mb-2">No remotes configured</h4>
										<p className="text-sm text-muted mb-6 max-w-md">
											Add a cloud storage remote to enable file browsing and automatic uploads after downloads complete.
										</p>
										<Button
											variant="primary"
											onPress={openWizard}
											className="rounded-xl font-bold"
										>
											<IconPlus className="w-4 h-4 mr-2" />
											Add Your First Remote
										</Button>
									</div>
								</Card>
							) : (
								<div className="space-y-3">
									{remotes.map((remote: any) => {
										const isDefault = defaultRemote.startsWith(`${remote.name}:`);
										return (
											<Card
												key={remote.name}
												className={cn(
													"p-5 transition-all",
													isDefault
														? "bg-accent/5 border-accent/30"
														: "bg-background/50 border-border"
												)}
											>
												<div className="flex items-center justify-between">
													<div className="flex items-center gap-4">
														<div className={cn(
															"w-12 h-12 rounded-2xl flex items-center justify-center text-xl",
															isDefault ? "bg-accent/10" : "bg-default/10"
														)}>
															{remote.type === "drive" ? "🔵" :
															 remote.type === "s3" ? "📦" :
															 remote.type === "dropbox" ? "🟦" :
															 remote.type === "onedrive" ? "🟢" :
															 remote.type === "local" ? "💾" : "📁"}
														</div>
														<div>
															<div className="flex items-center gap-2">
																<h4 className="font-bold text-base">{remote.name}</h4>
																{isDefault && (
																	<span className="flex items-center gap-1 text-[10px] font-black uppercase tracking-widest text-accent bg-accent/10 px-2 py-0.5 rounded-full">
																		<IconCheck className="w-3 h-3" />
																		Default
																	</span>
																)}
															</div>
															<p className="text-xs text-muted capitalize">{remote.type}</p>
														</div>
													</div>

													<div className="flex items-center gap-2">
														<Button
															size="sm"
															variant="ghost"
															onPress={() => navigate({ to: "/files", search: { path: `${remote.name}:` } })}
															className="rounded-xl font-bold"
														>
															<IconFolder className="w-4 h-4 mr-1" />
															Browse
														</Button>
														{!isDefault && (
															<Button
																size="sm"
																variant="ghost"
																onPress={() => handleSetDefault(remote.name)}
																className="rounded-xl font-bold"
															>
																Set Default
															</Button>
														)}
														<Button
															size="sm"
															variant="ghost"
															isIconOnly
															onPress={() => handleDelete(remote.name)}
															className="text-danger"
														>
															<IconTrashBin className="w-4 h-4" />
														</Button>
													</div>
												</div>
											</Card>
										);
									})}
								</div>
							)}
						</section>
					</div>
				</ScrollShadow>
			</div>

			{/* Add Remote Wizard Modal */}
			<Modal.Backdrop
				isOpen={isWizardOpen}
				onOpenChange={setIsWizardOpen}
				className="bg-background/80 backdrop-blur-sm"
			>
				<Modal.Container>
					<Modal.Dialog className="bg-surface border border-border shadow-2xl rounded-3xl w-full max-w-lg">
						<Modal.Header className="p-6 pb-2">
							<Modal.Heading className="text-xl font-bold">
								{wizard.step === "select" && "Add Cloud Remote"}
								{wizard.step === "configure" && `Configure ${wizard.provider?.name}`}
								{wizard.step === "success" && "Remote Added!"}
							</Modal.Heading>
						</Modal.Header>

						<Modal.Body className="px-6 py-4">
							{wizard.step === "select" && (
								<div className="space-y-4">
									<p className="text-sm text-muted">Select your cloud storage provider:</p>
									<div className="grid grid-cols-2 gap-3">
										{CLOUD_PROVIDERS.map((provider) => (
											<button
												key={provider.id}
												type="button"
												onClick={() => selectProvider(provider)}
												className={cn(
													"p-4 rounded-2xl border border-border hover:border-accent/50 transition-all text-left group",
													"hover:bg-accent/5"
												)}
											>
												<div className={cn(
													"w-10 h-10 rounded-xl flex items-center justify-center text-xl mb-2",
													provider.color
												)}>
													{provider.icon}
												</div>
												<p className="font-bold text-sm group-hover:text-accent transition-colors">
													{provider.name}
												</p>
											</button>
										))}
									</div>
								</div>
							)}

							{wizard.step === "configure" && wizard.provider && (
								<div className="space-y-6">
									<div>
										<Label className="text-sm font-bold mb-2 block">Remote Name</Label>
										<Input
											value={wizard.name}
											onChange={(e) => setWizard({ ...wizard, name: e.target.value })}
											placeholder={`my-${wizard.provider.id}`}
											className="h-12 bg-default/10 rounded-2xl border-none"
										/>
										<p className="text-xs text-muted mt-2">
											This is how you'll refer to this remote (e.g., {wizard.name || `my-${wizard.provider.id}`}:)
										</p>
									</div>

									<div className="h-px bg-border" />

									{renderConfigFields()}
								</div>
							)}

							{wizard.step === "success" && (
								<div className="text-center py-6">
									<div className="w-16 h-16 bg-success/10 rounded-full flex items-center justify-center mx-auto mb-4">
										<IconCheck className="w-8 h-8 text-success" />
									</div>
									<p className="text-lg font-bold mb-2">"{wizard.name}" is ready to use!</p>
									<p className="text-sm text-muted">
										You can now browse files and use this remote as an upload destination.
									</p>
								</div>
							)}
						</Modal.Body>

						<Modal.Footer className="p-6 pt-2 flex gap-2">
							{wizard.step === "select" && (
								<Button
									variant="ghost"
									className="flex-1 rounded-xl font-bold"
									onPress={closeWizard}
								>
									Cancel
								</Button>
							)}

							{wizard.step === "configure" && (
								<>
									<Button
										variant="ghost"
										className="flex-1 rounded-xl font-bold"
										onPress={() => setWizard({ ...wizard, step: "select" })}
									>
										← Back
									</Button>
									<Button
										variant="primary"
										className="flex-1 rounded-xl font-bold"
										onPress={handleCreate}
										isDisabled={!wizard.name}
										isPending={createRemote.isPending}
									>
										Connect
									</Button>
								</>
							)}

							{wizard.step === "success" && (
								<>
									<Button
										variant="ghost"
										className="flex-1 rounded-xl font-bold"
										onPress={() => {
											handleSetDefault(wizard.name);
											closeWizard();
										}}
									>
										Set as Default
									</Button>
									<Button
										variant="primary"
										className="flex-1 rounded-xl font-bold"
										onPress={() => {
											closeWizard();
											navigate({ to: "/files", search: { path: `${wizard.name}:` } });
										}}
									>
										Browse Files
									</Button>
								</>
							)}
						</Modal.Footer>
					</Modal.Dialog>
				</Modal.Container>
			</Modal.Backdrop>
		</div>
	);
}
