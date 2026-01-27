import { useEffect } from "react";
import { useQueryClient, type QueryClient } from "@tanstack/react-query";
import { openapi, subscribeEvents } from "../lib/openapi";
import type { components } from "../gen/api";

type Download = components["schemas"]["model.Download"];
type Stats = components["schemas"]["model.Stats"];
type EventResponse = components["schemas"]["api.EventResponse"];
type ProgressData = components["schemas"]["api.ProgressEventData"];

export function useDownloadEvents() {
  const queryClient = useQueryClient();

  useEffect(() => {
    const unsubscribe = subscribeEvents((payload) => {
      handleEvent(queryClient, payload);
    });

    return () => {
      unsubscribe();
    };
  }, [queryClient]);
}

function handleEvent(queryClient: QueryClient, event: EventResponse) {
  const { type, data } = event;

  switch (type) {
    case "download.progress":
      updateDownloadProgress(queryClient, data as ProgressData);
      break;
    case "upload.progress":
      updateUploadProgress(queryClient, data as ProgressData);
      break;
    case "stats":
      updateStats(queryClient, data as Stats);
      break;
    case "download.created":
    case "download.started":
    case "download.paused":
    case "download.resumed":
    case "download.completed":
    case "download.error":
    case "upload.started":
    case "upload.completed":
    case "upload.error": {
      const download = data as Download;
      if (download.id) {
        updateDownloadInList(queryClient, download);
      }
      // Invalidate using openapi keys
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/downloads").queryKey,
      });
      queryClient.invalidateQueries({
        queryKey: openapi.queryOptions("get", "/stats").queryKey,
      });
      if (download.id) {
        queryClient.invalidateQueries({
          queryKey: openapi.queryOptions("get", "/downloads/{id}", {
            params: { path: { id: download.id } },
          }).queryKey,
        });
      }
      break;
    }
  }
}

function updateDownloadProgress(
  queryClient: QueryClient,
  progressData: ProgressData,
) {
  const downloadsKey = openapi.queryOptions("get", "/downloads").queryKey;

  // Update List
  queryClient.setQueriesData<components["schemas"]["api.DownloadListResponse"]>(
    { queryKey: downloadsKey },
    (oldData) => {
      if (!oldData || !oldData.data) return oldData;

      return {
        ...oldData,
        data: oldData.data.map((d: Download) => {
          if (d.id === progressData.id) {
            return {
              ...d,
              downloaded: progressData.downloaded,
              size: progressData.size,
              speed: progressData.speed,
              eta: progressData.eta,
              seeders: progressData.seeders,
              peers: progressData.peers,
            };
          }
          return d;
        }),
      };
    },
  );

  // Update Detail Page
  const detailKey = openapi.queryOptions("get", "/downloads/{id}", {
    params: { path: { id: progressData.id } },
  }).queryKey;
  queryClient.setQueryData<components["schemas"]["api.DownloadResponse"]>(
    detailKey,
    (oldData) => {
      if (!oldData || !oldData.data) return oldData;

      return {
        ...oldData,
        data: {
          ...oldData.data,
          downloaded: progressData.downloaded,
          size: progressData.size,
          speed: progressData.speed,
          eta: progressData.eta,
          seeders: progressData.seeders,
          peers: progressData.peers,
        },
      };
    },
  );
}

function updateUploadProgress(
  queryClient: QueryClient,
  progressData: ProgressData,
) {
  const downloadsKey = openapi.queryOptions("get", "/downloads").queryKey;

  // Update List
  queryClient.setQueriesData<components["schemas"]["api.DownloadListResponse"]>(
    { queryKey: downloadsKey },
    (oldData) => {
      if (!oldData || !oldData.data) return oldData;

      return {
        ...oldData,
        data: oldData.data.map((d: Download) => {
          if (d.id === progressData.id) {
            return {
              ...d,
              uploadProgress:
                typeof progressData.uploaded === "number" &&
                progressData.size > 0
                  ? Math.floor(
                      (progressData.uploaded / progressData.size) * 100,
                    )
                  : d.uploadProgress,
              uploadSpeed: progressData.speed,
            };
          }
          return d;
        }),
      };
    },
  );

  // Update Detail Page
  const detailKey = openapi.queryOptions("get", "/downloads/{id}", {
    params: { path: { id: progressData.id } },
  }).queryKey;
  queryClient.setQueryData<components["schemas"]["api.DownloadResponse"]>(
    detailKey,
    (oldData) => {
      if (!oldData || !oldData.data) return oldData;

      return {
        ...oldData,
        data: {
          ...oldData.data,
          uploadProgress:
            typeof progressData.uploaded === "number" && progressData.size > 0
              ? Math.floor((progressData.uploaded / progressData.size) * 100)
              : oldData.data.uploadProgress,
          uploadSpeed: progressData.speed,
        },
      };
    },
  );
}

function updateStats(queryClient: QueryClient, statsData: Stats) {
  const statsKey = openapi.queryOptions("get", "/stats").queryKey;
  queryClient.setQueryData<components["schemas"]["api.StatsResponse"]>(
    statsKey,
    { data: statsData },
  );
}

function updateDownloadInList(
  queryClient: QueryClient,
  downloadData: Download,
) {
  if (!downloadData || !downloadData.id) return;

  const downloadsKey = openapi.queryOptions("get", "/downloads").queryKey;
  queryClient.setQueriesData<components["schemas"]["api.DownloadListResponse"]>(
    { queryKey: downloadsKey },
    (oldData) => {
      if (!oldData || !oldData.data) return oldData;

      const exists = oldData.data.find(
        (d: Download) => d.id === downloadData.id,
      );

      if (exists) {
        return {
          ...oldData,
          data: oldData.data.map((d: Download) =>
            d.id === downloadData.id ? { ...d, ...downloadData } : d,
          ),
        };
      }
      return oldData;
    },
  );

  // Update Detail Page
  const detailKey = openapi.queryOptions("get", "/downloads/{id}", {
    params: { path: { id: downloadData.id } },
  }).queryKey;
  queryClient.setQueryData<components["schemas"]["api.DownloadResponse"]>(
    detailKey,
    (oldData) => {
      if (!oldData || !oldData.data) return oldData;
      return { ...oldData, data: { ...oldData.data, ...downloadData } };
    },
  );
}
