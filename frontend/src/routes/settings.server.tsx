import { Alert, Button, Card, Chip, ScrollShadow, Spinner } from "@heroui/react";
import { createFileRoute } from "@tanstack/react-router";
import IconArrowRotateRight from "~icons/gravity-ui/arrow-rotate-right";
import IconCircleCheck from "~icons/gravity-ui/circle-check";
import IconServer from "~icons/gravity-ui/server";
import IconTriangleExclamation from "~icons/gravity-ui/triangle-exclamation";
import { useGravityVersion, useEngineActions } from "../hooks/useEngine";

export const Route = createFileRoute("/settings/server")({
	component: ServerManagement,
});

function ServerManagement() {
	const { data: systemInfo, isLoading } = useGravityVersion();
	const { restartAria2, restartRclone, restartServer } = useEngineActions();

	if (isLoading) {
		return (
			<div className="flex items-center justify-center h-full">
				<Spinner size="lg" color="accent" />
			</div>
		);
	}

	return (
		<div className="flex flex-col h-full space-y-6">
			<div className="flex items-center justify-between px-2 shrink-0">
				<div className="flex items-center gap-4">
					<h2 className="text-2xl font-bold tracking-tight">Server Management</h2>
					<Chip color="success" variant="soft">
						<IconCircleCheck className="w-3 h-3" />
						System Online
					</Chip>
				</div>
			</div>

			<div className="flex-1 bg-muted-background/40 rounded-3xl border border-border overflow-hidden min-h-0 mx-2">
				<ScrollShadow className="h-full">
					<div className="max-w-4xl mx-auto p-8 space-y-8">
						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-accent rounded-full" />
								<h3 className="text-lg font-bold">Engine Status</h3>
							</div>

							<div className="grid grid-cols-1 md:grid-cols-2 gap-6">
								{/* Aria2 Engine */}
								<Card className="bg-background/50 border-border">
									<Card.Header className="flex flex-row items-start justify-between">
										<div className="flex items-center gap-3">
											<div className="p-3 rounded-2xl bg-accent/10 text-accent">
												<IconServer className="w-6 h-6" />
											</div>
											<div className="flex flex-col">
												<Card.Title className="text-lg">aria2c</Card.Title>
												<Card.Description className="text-xs">Download Engine</Card.Description>
											</div>
										</div>
										<Chip size="sm" color="success" variant="soft">Running</Chip>
									</Card.Header>
									<Card.Content className="space-y-4">
										<div className="flex justify-between text-sm">
											<span className="text-muted">Version</span>
											<span className="font-mono">{systemInfo?.aria2 || "Unknown"}</span>
										</div>
										<Button
											fullWidth
											variant="secondary"
											className="font-bold rounded-xl"
											isPending={restartAria2.isPending}
											onPress={() => restartAria2.mutate()}
										>
											{({ isPending }) => (
												<>
													{isPending ? <Spinner size="sm" color="current" /> : <IconArrowRotateRight className="w-4 h-4" />}
													Restart Engine
												</>
											)}
										</Button>
									</Card.Content>
								</Card>

								{/* Rclone Engine */}
								<Card className="bg-background/50 border-border">
									<Card.Header className="flex flex-row items-start justify-between">
										<div className="flex items-center gap-3">
											<div className="p-3 rounded-2xl bg-cyan-500/10 text-cyan-500">
												<IconServer className="w-6 h-6" />
											</div>
											<div className="flex flex-col">
												<Card.Title className="text-lg">rclone</Card.Title>
												<Card.Description className="text-xs">Upload & File Engine</Card.Description>
											</div>
										</div>
										<Chip size="sm" color="success" variant="soft">Running</Chip>
									</Card.Header>
									<Card.Content className="space-y-4">
										<div className="flex justify-between text-sm">
											<span className="text-muted">Version</span>
											<span className="font-mono">{systemInfo?.rclone || "Unknown"}</span>
										</div>
										<Button
											fullWidth
											variant="secondary"
											className="font-bold rounded-xl"
											isPending={restartRclone.isPending}
											onPress={() => restartRclone.mutate()}
										>
											{({ isPending }) => (
												<>
													{isPending ? <Spinner size="sm" color="current" /> : <IconArrowRotateRight className="w-4 h-4" />}
													Restart Engine
												</>
											)}
										</Button>
									</Card.Content>
								</Card>
							</div>
						</section>

						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-danger rounded-full" />
								<h3 className="text-lg font-bold">System Maintenance</h3>
							</div>

							<Card className="border-danger/20 bg-danger/5">
								<Card.Content className="p-6">
									<div className="flex flex-col md:flex-row items-center justify-between gap-6">
										<div className="flex items-center gap-4">
											<div className="p-3 rounded-2xl bg-danger/10 text-danger">
												<IconTriangleExclamation className="w-6 h-6" />
											</div>
											<div>
												<h4 className="font-bold">Restart Gravity Server</h4>
												<p className="text-sm text-muted">
													This will restart the main server process. All active connections will be dropped.
												</p>
											</div>
										</div>
										<Button
											variant="primary"
											className="font-bold rounded-xl px-8"
											isPending={restartServer.isPending}
											onPress={() => {
												if (confirm("Are you sure you want to restart the server?")) {
													restartServer.mutate();
												}
											}}
										>
											{({ isPending }) => (
												<>
													{isPending ? <Spinner size="sm" color="current" /> : <IconArrowRotateRight className="w-4 h-4" />}
													Restart Server
												</>
											)}
										</Button>
									</div>
								</Card.Content>
							</Card>

							<div className="h-4" />

							<Alert status="warning">
								<Alert.Indicator />
								<Alert.Content>
									<Alert.Title>Self-Restart Notice</Alert.Title>
									<Alert.Description>
										Gravity requires a process manager (like Docker with restart policies or Systemd) to automatically restart the process after it exits.
									</Alert.Description>
								</Alert.Content>
							</Alert>
						</section>

						<section>
							<div className="flex items-center gap-3 mb-6">
								<div className="w-1.5 h-6 bg-muted rounded-full" />
								<h3 className="text-lg font-bold">Build Information</h3>
							</div>
							<Card className="bg-background/50 border-border">
								<Card.Content className="p-6">
									<div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-12">
										<div className="flex justify-between items-center py-2 border-b border-border">
											<span className="text-sm text-muted">Gravity Version</span>
											<span className="text-sm font-bold">{systemInfo?.version || "0.1.0"}</span>
										</div>
										<div className="flex justify-between items-center py-2 border-b border-border">
											<span className="text-sm text-muted">Frontend Version</span>
											<span className="text-sm font-bold">1.0.0</span>
										</div>
										<div className="flex justify-between items-center py-2 border-b border-border">
											<span className="text-sm text-muted">Environment</span>
											<span className="text-sm font-bold capitalize">Production</span>
										</div>
										<div className="flex justify-between items-center py-2 border-b border-border">
											<span className="text-sm text-muted">OS/Arch</span>
											<span className="text-sm font-bold">Linux/amd64</span>
										</div>
									</div>
								</Card.Content>
							</Card>
						</section>
					</div>
				</ScrollShadow>
			</div>
		</div>
	);
}
