import { Button, Card, ScrollShadow } from "@heroui/react";
import { createFileRoute, Link } from "@tanstack/react-router";
import IconRocket from "~icons/gravity-ui/rocket";
import IconCloud from "~icons/gravity-ui/cloud";
import IconThunderbolt from "~icons/gravity-ui/thunderbolt";
import IconGlobe from "~icons/gravity-ui/globe";
import IconMagnet from "~icons/gravity-ui/magnet";
import IconGear from "~icons/gravity-ui/gear";
import IconCheck from "~icons/gravity-ui/check";
import IconCircleExclamation from "~icons/gravity-ui/circle-exclamation";
import IconFolder from "~icons/gravity-ui/folder";
import IconCloudArrowUpIn from "~icons/gravity-ui/cloud-arrow-up-in";
import { useRemotes } from "../hooks/useRemotes";
import { useProviders } from "../hooks/useProviders";
import { useSettingsStore } from "../store/useSettingsStore";
import { cn } from "../lib/utils";

export const Route = createFileRoute("/settings/")({
	component: SettingsOverview,
});

interface SetupCardProps {
	title: string;
	description: string;
	icon: React.ReactNode;
	status: "configured" | "needs-setup" | "optional";
	to: string;
	statusText?: string;
}

function SetupCard({ title, description, icon, status, to, statusText }: SetupCardProps) {
	return (
		<Link to={to} className="block group">
			<Card className={cn(
				"p-5 h-full transition-all duration-200 hover:shadow-md",
				status === "configured" && "border-success/30 bg-success/5",
				status === "needs-setup" && "border-warning/30 bg-warning/5",
				status === "optional" && "border-border bg-default/5",
			)}>
				<div className="flex items-start gap-4">
					<div className={cn(
						"p-3 rounded-2xl shrink-0",
						status === "configured" && "bg-success/10 text-success",
						status === "needs-setup" && "bg-warning/10 text-warning",
						status === "optional" && "bg-default/10 text-muted",
					)}>
						{icon}
					</div>
					<div className="flex-1 min-w-0">
						<div className="flex items-center gap-2 mb-1">
							<h3 className="font-bold text-base group-hover:text-accent transition-colors">{title}</h3>
							{status === "configured" && (
								<span className="flex items-center gap-1 text-[10px] font-black uppercase tracking-widest text-success bg-success/10 px-2 py-0.5 rounded-full">
									<IconCheck className="w-3 h-3" />
									{statusText || "Ready"}
								</span>
							)}
							{status === "needs-setup" && (
								<span className="flex items-center gap-1 text-[10px] font-black uppercase tracking-widest text-warning bg-warning/10 px-2 py-0.5 rounded-full">
									<IconCircleExclamation className="w-3 h-3" />
									Setup
								</span>
							)}
						</div>
						<p className="text-xs text-muted line-clamp-2">{description}</p>
					</div>
				</div>
			</Card>
		</Link>
	);
}

interface SettingsLinkProps {
	title: string;
	description: string;
	icon: React.ReactNode;
	to: string;
}

function SettingsLink({ title, description, icon, to }: SettingsLinkProps) {
	return (
		<Link to={to} className="block group">
			<div className="flex items-center gap-4 p-4 rounded-2xl hover:bg-default/10 transition-colors">
				<div className="p-2.5 rounded-xl bg-default/10 text-muted group-hover:bg-accent/10 group-hover:text-accent transition-colors">
					{icon}
				</div>
				<div className="flex-1 min-w-0">
					<h4 className="font-bold text-sm group-hover:text-accent transition-colors">{title}</h4>
					<p className="text-xs text-muted">{description}</p>
				</div>
			</div>
		</Link>
	);
}

function SettingsOverview() {
	const { data: remotes = [] } = useRemotes();
	const { data: providersResponse } = useProviders();
	const { defaultRemote } = useSettingsStore();

	const providers = providersResponse?.data || [];

	const hasRemotes = remotes.length > 0;
	const connectedProviders = providers.filter((p: any) => p.enabled);
	const hasProviders = connectedProviders.length > 0;

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex items-center gap-4 px-2 shrink-0">
				<h2 className="text-2xl font-bold tracking-tight">Settings</h2>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{/* Setup Status */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Setup Status</h3>
							</div>
							<div className="grid grid-cols-1 md:grid-cols-2 gap-4">
								<SetupCard
									title="Downloads"
									description="Configure download speed, queue limits, and storage location"
									icon={<IconRocket className="w-5 h-5" />}
									status="configured"
									statusText="Configured"
									to="/settings/downloads"
								/>
								<SetupCard
									title="Cloud Storage"
									description={hasRemotes 
										? `${remotes.length} remote${remotes.length > 1 ? 's' : ''} configured`
										: "Set up cloud remotes for file browsing and auto-upload"
									}
									icon={<IconCloud className="w-5 h-5" />}
									status={hasRemotes ? "configured" : "needs-setup"}
									statusText={hasRemotes ? `${remotes.length} remote${remotes.length > 1 ? 's' : ''}` : undefined}
									to="/settings/cloud"
								/>
								<SetupCard
									title="Premium Services"
									description={hasProviders 
										? `${connectedProviders.length} service${connectedProviders.length > 1 ? 's' : ''} connected`
										: "Connect AllDebrid, Real-Debrid, or other debrid services"
									}
									icon={<IconThunderbolt className="w-5 h-5" />}
									status={hasProviders ? "configured" : "optional"}
									statusText={hasProviders ? "Connected" : undefined}
									to="/settings/premium"
								/>
								<SetupCard
									title="Network"
									description="Proxy settings, connection limits, and security options"
									icon={<IconGlobe className="w-5 h-5" />}
									status="optional"
									to="/settings/network"
								/>
								<SetupCard
									title="Browser"
									description="File browser cache duration and display settings"
									icon={<IconFolder className="w-5 h-5" />}
									status="optional"
									to="/settings/browser"
								/>
							</div>
						</section>

						{/* Quick Settings */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Quick Settings</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border">
								<div className="space-y-5">
									<div className="flex items-center justify-between py-2">
										<div className="flex items-center gap-3">
											<div className="p-2 rounded-lg bg-default/10">
												<IconFolder className="w-4 h-4 text-muted" />
											</div>
											<div>
												<p className="text-sm font-bold">Download Location</p>
												<p className="text-xs text-muted">Where files are saved on the server</p>
											</div>
										</div>
										<Link to="/settings/downloads">
											<Button size="sm" variant="secondary" className="rounded-xl font-bold">
												Configure
											</Button>
										</Link>
									</div>
									
									<div className="h-px bg-border" />
									
									<div className="flex items-center justify-between py-2">
										<div className="flex items-center gap-3">
											<div className="p-2 rounded-lg bg-default/10">
												<IconCloudArrowUpIn className="w-4 h-4 text-muted" />
											</div>
											<div>
												<p className="text-sm font-bold">Auto-Upload Destination</p>
												<p className="text-xs text-muted">
													{defaultRemote || "Not configured - files stay local after download"}
												</p>
											</div>
										</div>
										<Link to="/settings/cloud">
											<Button size="sm" variant={defaultRemote ? "secondary" : "primary"} className="rounded-xl font-bold">
												{defaultRemote ? "Change" : "Set Up"}
											</Button>
										</Link>
									</div>
									
									<div className="h-px bg-border" />
									
									<div className="flex items-center justify-between py-2">
										<div className="flex items-center gap-3">
											<div className="p-2 rounded-lg bg-default/10">
												<IconThunderbolt className="w-4 h-4 text-muted" />
											</div>
											<div>
												<p className="text-sm font-bold">Premium Downloads</p>
												<p className="text-xs text-muted">
													{hasProviders 
														? `Using ${connectedProviders[0]?.name || 'premium service'} for supported links`
														: "Connect a debrid service for faster downloads"
													}
												</p>
											</div>
										</div>
										<Link to="/settings/premium">
											<Button size="sm" variant={hasProviders ? "secondary" : "ghost"} className="rounded-xl font-bold">
												{hasProviders ? "Manage" : "Connect"}
											</Button>
										</Link>
									</div>
								</div>
							</Card>
						</section>

						{/* All Settings */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">All Settings</h3>
							</div>
							<Card className="bg-background/50 border-border overflow-hidden">
								<SettingsLink
									title="Downloads"
									description="Speed limits, queue management, storage, automation"
									icon={<IconRocket className="w-4 h-4" />}
									to="/settings/downloads"
								/>
								<div className="h-px bg-border mx-4" />
								<SettingsLink
									title="Cloud Storage"
									description="Manage remotes and upload destinations"
									icon={<IconCloud className="w-4 h-4" />}
									to="/settings/cloud"
								/>
								<div className="h-px bg-border mx-4" />
								<SettingsLink
									title="Premium Services"
									description="AllDebrid, Real-Debrid, and other debrid providers"
									icon={<IconThunderbolt className="w-4 h-4" />}
									to="/settings/premium"
								/>
								<div className="h-px bg-border mx-4" />
								<SettingsLink
									title="Network"
									description="Proxy, connections, security, advanced networking"
									icon={<IconGlobe className="w-4 h-4" />}
									to="/settings/network"
								/>
								<div className="h-px bg-border mx-4" />
								<SettingsLink
									title="Torrents"
									description="BitTorrent seeding, privacy, trackers"
									icon={<IconMagnet className="w-4 h-4" />}
									to="/settings/torrents"
								/>
								<div className="h-px bg-border mx-4" />
								<SettingsLink
									title="Preferences"
									description="Theme, notifications, keyboard shortcuts"
									icon={<IconGear className="w-4 h-4" />}
									to="/settings/preferences"
								/>
								<div className="h-px bg-border mx-4" />
								<SettingsLink
									title="Browser"
									description="File browser cache duration and display"
									icon={<IconFolder className="w-4 h-4" />}
									to="/settings/browser"
								/>
								<div className="h-px bg-border mx-4" />
								<SettingsLink
									title="Server"
									description="Engine versions, system status, and maintenance"
									icon={<IconGear className="w-4 h-4" />}
									to="/settings/server"
								/>
							</Card>
						</section>
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}
