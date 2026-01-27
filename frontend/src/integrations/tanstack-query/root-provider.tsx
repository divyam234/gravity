import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ApiError } from "../../lib/openapi";

const MAX_RETRIES = 3;
const HTTP_STATUS_TO_NOT_RETRY = [401, 403, 404];

export function getContext() {
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				gcTime: 10 * (60 * 1000),
				staleTime: 5 * (60 * 1000),
				retry: (failureCount, error) => {
					if (failureCount >= MAX_RETRIES) {
						return false;
					}

					if (error instanceof ApiError && HTTP_STATUS_TO_NOT_RETRY.includes(error.status)) {
						return false;
					}

					return true;
				},
			},
		},
	});
	return {
		queryClient,
	};
}

export function Provider({
	children,
	queryClient,
}: {
	children: React.ReactNode;
	queryClient: QueryClient;
}) {
	return (
		<QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
	);
}
