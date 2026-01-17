import { Button, Card, Input, Label } from "@heroui/react";
import { TanStackDevtools } from "@tanstack/react-devtools";
import type { QueryClient } from "@tanstack/react-query";
import { useQuery } from "@tanstack/react-query";
import {
	createRootRouteWithContext,
	Outlet,
	useRouter,
	useRouterState,
} from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import React from "react";
import LoadingBar from "react-top-loading-bar";
import IconGear from "~icons/gravity-ui/gear";
import IconLink from "~icons/gravity-ui/link";
import IconShieldKeyhole from "~icons/gravity-ui/shield-keyhole";
import { Layout } from "../components/Layout";
import { globalStatOptions } from "../hooks/useAria2";
import TanStackQueryDevtools from "../integrations/tanstack-query/devtools";
import { useSettingsStore } from "../store/useSettingsStore";

interface MyRouterContext {
	queryClient: QueryClient;
}

export const Route = createRootRouteWithContext<MyRouterContext>()({
	component: RootComponent,
});

interface ConnectionErrorOverlayProps {
	refetch: () => void;
}

function ConnectionErrorOverlay({ refetch }: ConnectionErrorOverlayProps) {
	const { rpcUrl, rpcSecret, setRpcUrl, setRpcSecret } = useSettingsStore();
	const [localUrl, setLocalUrl] = React.useState(rpcUrl);
	const [localSecret, setLocalSecret] = React.useState(rpcSecret);
	const router = useRouter();
	const urlId = React.useId();
	const secretId = React.useId();

	const handleSave = async () => {
		setRpcUrl(localUrl);
		setRpcSecret(localSecret);
		// Let the sync effect run and the query refetch
		setTimeout(async () => {
			refetch();
			await router.invalidate();
		}, 100);
	};

	return (
		<div className="fixed inset-0 z-[9999] bg-background/80 backdrop-blur-xl flex items-center justify-center p-4">
			<Card className="max-w-md w-full p-8 shadow-2xl border-border animate-in fade-in zoom-in duration-300">
				<div className="flex flex-col items-center text-center gap-6">
					<div className="w-16 h-16 rounded-2xl bg-danger/10 flex items-center justify-center text-danger shadow-glow-lg shadow-danger/20">
						<IconGear className="w-8 h-8 animate-spin-slow" />
					</div>

					<div className="space-y-2">
						<h1 className="text-2xl font-bold tracking-tight">
							{rpcUrl ? "Connection Failed" : "Welcome to aria2c"}
						</h1>
						<p className="text-muted text-sm">
							{rpcUrl
								? "Could not connect to aria2 RPC. Please check your settings."
								: "Please configure your aria2 RPC endpoint to get started."}
						</p>
					</div>

					<div className="w-full space-y-4">
						<div className="space-y-2 text-left">
							<Label
								htmlFor={urlId}
								className="text-xs font-black uppercase tracking-widest text-muted ml-1"
							>
								RPC Endpoint
							</Label>
							<div className="relative">
								<IconLink className="absolute left-3 top-1/2 -translate-y-1/2 text-muted w-4 h-4 pointer-events-none z-10" />
								<Input
									id={urlId}
									placeholder="http://localhost:6800/jsonrpc"
									value={localUrl}
									onChange={(e) => setLocalUrl(e.target.value)}
									className="font-mono text-sm pl-10 h-11 border border-border"
									fullWidth
								/>
							</div>
						</div>

						<div className="space-y-2 text-left">
							<Label
								htmlFor={secretId}
								className="text-xs font-black uppercase tracking-widest text-muted ml-1"
							>
								Secret Token (Optional)
							</Label>
							<div className="relative">
								<IconShieldKeyhole className="absolute left-3 top-1/2 -translate-y-1/2 text-muted w-4 h-4 pointer-events-none z-10" />
								<Input
									id={secretId}
									type="password"
									placeholder="Enter RPC Secret"
									value={localSecret}
									onChange={(e) => setLocalSecret(e.target.value)}
									className="font-mono text-sm pl-10 h-11 border border-border"
									fullWidth
								/>
							</div>
						</div>
					</div>

					<Button
						variant="primary"
						className="w-full font-bold h-12 rounded-xl hover-glow shadow-lg shadow-accent/20 bg-accent text-accent-foreground"
						onPress={handleSave}
					>
						Reconnect & Save
					</Button>

					<p className="text-[10px] text-muted uppercase tracking-tighter">
						Tip: Ensure aria2c is running with --enable-rpc
					</p>
				</div>
			</Card>
		</div>
	);
}

function RootComponent() {
	const isLoading = useRouterState({ select: (s) => s.status === "pending" });
	const [progress, setProgress] = React.useState(0);
	const { rpcUrl, pollingInterval } = useSettingsStore();

	const globalStat = useQuery({
		...globalStatOptions(rpcUrl, pollingInterval),
		enabled: !!rpcUrl,
	});

	const showOverlay = !rpcUrl || globalStat.isError;

	React.useEffect(() => {
		if (isLoading) {
			setProgress(40);
		} else {
			setProgress(100);
		}
	}, [isLoading]);

	return (
		<Layout>
			<LoadingBar
				color="var(--accent)"
				progress={progress}
				onLoaderFinished={() => setProgress(0)}
				height={3}
			/>
			{showOverlay ? (
				<ConnectionErrorOverlay refetch={globalStat.refetch} />
			) : (
				<Outlet />
			)}
			<TanStackDevtools
				config={{
					position: "bottom-right",
				}}
				plugins={[
					{
						name: "Tanstack Router",
						render: <TanStackRouterDevtoolsPanel />,
					},
					TanStackQueryDevtools,
				]}
			/>
		</Layout>
	);
}
