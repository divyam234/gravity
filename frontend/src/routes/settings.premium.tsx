import { Button, Card, Input, Label, Modal, ScrollShadow, Switch, Spinner } from "@heroui/react";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import IconChevronLeft from "~icons/gravity-ui/chevron-left";
import IconCheck from "~icons/gravity-ui/check";
import IconThunderbolt from "~icons/gravity-ui/thunderbolt";
import IconArrowUpRightFromSquare from "~icons/gravity-ui/arrow-up-right-from-square";
import { useProviders, useProviderActions } from "../hooks/useProviders";
import { cn } from "../lib/utils";

export const Route = createFileRoute("/settings/premium")({
	component: PremiumServicesPage,
});

// Premium provider definitions
const PREMIUM_PROVIDERS = [
	{
		id: "alldebrid",
		name: "AllDebrid",
		description: "Fast debrid service with 80+ hosts and instant torrent caching",
		color: "bg-green-500/10 text-green-500 border-green-500/20",
		icon: "AD",
		website: "https://alldebrid.com",
		configFields: [{ key: "api_key", label: "API Key", type: "password" }],
	},
	{
		id: "realdebrid",
		name: "Real-Debrid",
		description: "Premium link generator with 100+ hosts support",
		color: "bg-red-500/10 text-red-500 border-red-500/20",
		icon: "RD",
		website: "https://real-debrid.com",
		configFields: [{ key: "api_key", label: "API Key", type: "password" }],
	},
	{
		id: "premiumize",
		name: "Premiumize",
		description: "Cloud downloader with VPN and Usenet support",
		color: "bg-orange-500/10 text-orange-500 border-orange-500/20",
		icon: "PM",
		website: "https://premiumize.me",
		configFields: [{ key: "api_key", label: "API Key", type: "password" }],
	},
	{
		id: "debridlink",
		name: "Debrid-Link",
		description: "European debrid service with competitive pricing",
		color: "bg-blue-500/10 text-blue-500 border-blue-500/20",
		icon: "DL",
		website: "https://debrid-link.com",
		configFields: [{ key: "api_key", label: "API Key", type: "password" }],
	},
	{
		id: "torbox",
		name: "TorBox",
		description: "All-in-one torrent, Usenet, and debrid service",
		color: "bg-purple-500/10 text-purple-500 border-purple-500/20",
		icon: "TB",
		website: "https://torbox.app",
		configFields: [{ key: "api_key", label: "API Key", type: "password" }],
	},
];

interface ConnectModalState {
	isOpen: boolean;
	provider: typeof PREMIUM_PROVIDERS[number] | null;
	config: Record<string, string>;
}

function PremiumServicesPage() {
	const navigate = useNavigate();
	const { data: providersResponse, isLoading } = useProviders();
	const { configure } = useProviderActions();

	const providers = providersResponse?.data || [];

	const [connectModal, setConnectModal] = useState<ConnectModalState>({
		isOpen: false,
		provider: null,
		config: {},
	});

	// Preferences (these would ideally be stored in settings)
	const [usePremium, setUsePremium] = useState(true);
	const [usePremiumForMagnets, setUsePremiumForMagnets] = useState(true);
	const [fallbackToDirect, setFallbackToDirect] = useState(true);

	const openConnectModal = (provider: typeof PREMIUM_PROVIDERS[number]) => {
		setConnectModal({
			isOpen: true,
			provider,
			config: {},
		});
	};

	const closeConnectModal = () => {
		setConnectModal({ isOpen: false, provider: null, config: {} });
	};

	const handleConnect = async () => {
		if (!connectModal.provider) return;

		configure.mutate({
			name: connectModal.provider.id,
			config: connectModal.config,
			enabled: true,
		}, {
			onSuccess: () => closeConnectModal(),
		});
	};

	const handleDisconnect = (providerId: string) => {
		if (confirm("Disconnect this service? You'll need to re-enter your API key to use it again.")) {
			configure.mutate({
				name: providerId,
				config: {},
				enabled: false,
			});
		}
	};

	const connectedProviders = providers.filter((p: any) => p.enabled);

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
					<h2 className="text-2xl font-bold tracking-tight">Premium Services</h2>
					<p className="text-xs text-muted">Debrid providers & link resolvers</p>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-10">
						{/* Info Banner */}
						<Card className="p-5 bg-accent/5 border-accent/20">
							<div className="flex items-start gap-4">
								<div className="p-2.5 rounded-xl bg-accent/10">
									<IconThunderbolt className="w-5 h-5 text-accent" />
								</div>
								<div>
									<h4 className="font-bold mb-1">What are Premium Services?</h4>
									<p className="text-sm text-muted">
										Premium services like AllDebrid and Real-Debrid unlock high-speed downloads from 100+ file hosts
										and provide instant torrent/magnet caching - no waiting for peers!
									</p>
								</div>
							</div>
						</Card>

						{/* Preferences */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Download Preferences</h3>
							</div>
							<Card className="p-6 bg-background/50 border-border space-y-5">
								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Use premium services when available</Label>
										<p className="text-xs text-muted mt-0.5">Automatically use debrid for supported links</p>
									</div>
									<Switch isSelected={usePremium} onChange={setUsePremium}>
										<Switch.Control>
											<Switch.Thumb />
										</Switch.Control>
									</Switch>
								</div>

								<div className="h-px bg-border" />

								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Use premium for magnets & torrents</Label>
										<p className="text-xs text-muted mt-0.5">Download cached torrents via debrid instead of P2P</p>
									</div>
									<Switch isSelected={usePremiumForMagnets} onChange={setUsePremiumForMagnets}>
										<Switch.Control>
											<Switch.Thumb />
										</Switch.Control>
									</Switch>
								</div>

								<div className="h-px bg-border" />

								<div className="flex items-center justify-between">
									<div>
										<Label className="text-sm font-bold">Fallback to direct download</Label>
										<p className="text-xs text-muted mt-0.5">Use direct/P2P download if premium fails</p>
									</div>
									<Switch isSelected={fallbackToDirect} onChange={setFallbackToDirect}>
										<Switch.Control>
											<Switch.Thumb />
										</Switch.Control>
									</Switch>
								</div>
							</Card>
						</section>

						{/* Connected Services */}
						{connectedProviders.length > 0 && (
							<section>
								<div className="flex items-center gap-3 mb-6">
									<div className="w-1.5 h-6 bg-success rounded-full" />
									<h3 className="text-lg font-bold">Connected Services</h3>
								</div>
								<div className="space-y-3">
									{connectedProviders.map((provider: any) => {
										const def = PREMIUM_PROVIDERS.find(p => p.id === provider.name || p.id === provider.id);
										if (!def) return null;

										return (
											<Card key={def.id} className={cn("p-5", def.color, "border")}>
												<div className="flex items-center justify-between">
													<div className="flex items-center gap-4">
														<div className="w-12 h-12 rounded-2xl bg-background/50 flex items-center justify-center font-black text-lg">
															{def.icon}
														</div>
														<div>
															<div className="flex items-center gap-2">
																<h4 className="font-bold text-base">{def.name}</h4>
																<span className="flex items-center gap-1 text-[10px] font-black uppercase tracking-widest text-success bg-success/10 px-2 py-0.5 rounded-full">
																	<IconCheck className="w-3 h-3" />
																	Connected
																</span>
															</div>
															{provider.expiresAt && (
																<p className="text-xs text-muted mt-0.5">
																	Premium until: {new Date(provider.expiresAt).toLocaleDateString()}
																</p>
															)}
														</div>
													</div>

													<div className="flex items-center gap-2">
														<Button
															size="sm"
															variant="ghost"
															onPress={() => handleDisconnect(def.id)}
															className="rounded-xl font-bold text-danger"
														>
															Disconnect
														</Button>
													</div>
												</div>

												{/* Usage stats if available */}
												{provider.usage && (
													<div className="mt-4 pt-4 border-t border-current/10 grid grid-cols-2 gap-4">
														<div>
															<p className="text-[10px] font-black uppercase tracking-widest opacity-70">Magnets Today</p>
															<p className="font-bold">{provider.usage.magnets?.used || 0} / {provider.usage.magnets?.limit || '∞'}</p>
														</div>
														<div>
															<p className="text-[10px] font-black uppercase tracking-widest opacity-70">Links Today</p>
															<p className="font-bold">{provider.usage.links?.used || 0} / {provider.usage.links?.limit || '∞'}</p>
														</div>
													</div>
												)}
											</Card>
										);
									})}
								</div>
							</section>
						)}

						{/* Available Services */}
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">
									{connectedProviders.length > 0 ? "Add More Services" : "Available Services"}
								</h3>
							</div>
							
							{isLoading ? (
								<div className="flex justify-center py-12">
									<Spinner size="md" />
								</div>
							) : (
								<div className="space-y-3">
									{PREMIUM_PROVIDERS
										.filter(def => !connectedProviders.some((p: any) => p.name === def.id || p.id === def.id))
										.map((provider) => (
											<Card key={provider.id} className="p-5 bg-background/50 border-border hover:border-accent/30 transition-colors">
												<div className="flex items-center justify-between">
													<div className="flex items-center gap-4">
														<div className={cn(
															"w-12 h-12 rounded-2xl flex items-center justify-center font-black text-lg",
															provider.color
														)}>
															{provider.icon}
														</div>
														<div>
															<h4 className="font-bold text-base">{provider.name}</h4>
															<p className="text-xs text-muted mt-0.5 max-w-md">{provider.description}</p>
														</div>
													</div>

													<div className="flex items-center gap-2">
														<Button
															size="sm"
															variant="ghost"
															onPress={() => window.open(provider.website, '_blank')}
															className="rounded-xl"
															isIconOnly
														>
															<IconArrowUpRightFromSquare className="w-4 h-4" />
														</Button>
														<Button
															size="sm"
															variant="primary"
															onPress={() => openConnectModal(provider)}
															className="rounded-xl font-bold"
														>
															Connect
														</Button>
													</div>
												</div>
											</Card>
										))}
								</div>
							)}
						</section>
					</div>
				</ScrollShadow>
			</div>

			{/* Connect Modal */}
			<Modal.Backdrop
				isOpen={connectModal.isOpen}
				onOpenChange={(open) => !open && closeConnectModal()}
				className="bg-background/80 backdrop-blur-sm"
			>
				<Modal.Container>
					<Modal.Dialog className="bg-content1 border border-border shadow-2xl rounded-3xl w-full max-w-md">
						<Modal.Header className="p-6 pb-2">
							<Modal.Heading className="text-xl font-bold">
								Connect {connectModal.provider?.name}
							</Modal.Heading>
						</Modal.Header>

						<Modal.Body className="px-6 py-4 space-y-6">
							<p className="text-sm text-muted">
								Enter your API key to connect your {connectModal.provider?.name} account.
							</p>

							{connectModal.provider?.configFields.map((field) => (
								<div key={field.key}>
									<Label className="text-sm font-bold mb-2 block">{field.label}</Label>
									<Input
										type={field.type}
										value={connectModal.config[field.key] || ""}
										onChange={(e) => setConnectModal({
											...connectModal,
											config: { ...connectModal.config, [field.key]: e.target.value }
										})}
										placeholder="Paste your API key here..."
										className="h-12 bg-default/10 rounded-2xl border-none"
									/>
								</div>
							))}

							<Card className="p-4 bg-default/5 border-border">
								<p className="text-xs text-muted mb-2">💡 How to get your API key:</p>
								<ol className="text-xs text-muted space-y-1 list-decimal list-inside">
									<li>Go to {connectModal.provider?.website}</li>
									<li>Log in to your account</li>
									<li>Navigate to Account/API settings</li>
									<li>Copy your API key or token</li>
								</ol>
								<Button
									size="sm"
									variant="ghost"
									onPress={() => window.open(connectModal.provider?.website, '_blank')}
									className="mt-3 text-accent font-bold"
								>
									Open {connectModal.provider?.name} <IconArrowUpRightFromSquare className="w-3 h-3 ml-1" />
								</Button>
							</Card>
						</Modal.Body>

						<Modal.Footer className="p-6 pt-2 flex gap-2">
							<Button
								variant="ghost"
								className="flex-1 rounded-xl font-bold"
								onPress={closeConnectModal}
							>
								Cancel
							</Button>
							<Button
								variant="primary"
								className="flex-1 rounded-xl font-bold"
								onPress={handleConnect}
								isDisabled={!connectModal.config.api_key}
								isPending={configure.isPending}
							>
								Connect
							</Button>
						</Modal.Footer>
					</Modal.Dialog>
				</Modal.Container>
			</Modal.Backdrop>
		</div>
	);
}
