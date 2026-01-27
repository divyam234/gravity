import createClient, { type Middleware } from "openapi-fetch";
import createQueryClient from "openapi-react-query";
import type { paths, components } from "../gen/api";

let currentApiKey = "";
let currentBaseUrl = "/api/v1";

const middleware: Middleware = {
  onRequest: ({ request }) => {
    if (currentApiKey) {
      request.headers.set("X-API-Key", currentApiKey);
    }
    return request;
  },
};

export const client = createClient<paths>({
  baseUrl: "/api/v1",
});

client.use(middleware);

export const openapi = createQueryClient(client);

export function configureClient(baseUrl: string, apiKey: string) {
  currentBaseUrl = baseUrl;
  currentApiKey = apiKey;
}

export function getBaseUrl() {
  return currentBaseUrl;
}

export function getFileUrl(path: string) {
  return `${currentBaseUrl}/files/cat?path=${encodeURIComponent(path)}`;
}

export function subscribeEvents(handler: (event: components["schemas"]["api.EventResponse"]) => void) {
  const evtSource = new EventSource(`${currentBaseUrl}/events`);
  evtSource.onmessage = (e) => handler(JSON.parse(e.data));
  return () => evtSource.close();
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}