import { TanStackDevtools } from "@tanstack/react-devtools";
import type { QueryClient } from "@tanstack/react-query";
import {
	createRootRouteWithContext,
	Outlet,
	useRouterState,
} from "@tanstack/react-router";
import { TanStackRouterDevtoolsPanel } from "@tanstack/react-router-devtools";
import React from "react";
import LoadingBar from "react-top-loading-bar";
import { Layout } from "../components/Layout";
import TanStackQueryDevtools from "../integrations/tanstack-query/devtools";

interface MyRouterContext {
	queryClient: QueryClient;
}

export const Route = createRootRouteWithContext<MyRouterContext>()({
	component: RootComponent,
});

function RootComponent() {
	const isLoading = useRouterState({ select: (s) => s.status === "pending" });
	const [progress, setProgress] = React.useState(0);

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
				color="oklch(var(--color-primary))"
				progress={progress}
				onLoaderFinished={() => setProgress(0)}
				height={3}
			/>
			<Outlet />
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
