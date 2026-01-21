import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import type { Download } from '../lib/types';

interface DownloadEvent {
  type: string;
  timestamp: string;
  data: any;
}

export function useDownloadEvents() {
  const queryClient = useQueryClient();
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const unsubscribe = api.subscribeEvents((payload: DownloadEvent) => {
      handleEvent(queryClient, payload);
    });

    return () => {
      unsubscribe();
    };
  }, [queryClient]);
}

function handleEvent(queryClient: any, event: DownloadEvent) {
  const { type, data } = event;

  switch (type) {
    case 'download.progress':
      updateDownloadProgress(queryClient, data);
      break;
    case 'upload.progress':
      updateUploadProgress(queryClient, data);
      break;
    case 'stats':
      updateStats(queryClient, data);
      break;
    case 'download.created':
    case 'download.started':
    case 'download.paused':
    case 'download.resumed':
    case 'download.completed':
    case 'download.error':
    case 'upload.started':
    case 'upload.completed':
    case 'upload.error':
      // For significant state changes, we might want to refetch the list
      // Or optimistically update if we have the full object
      if (data.id) {
         // Ideally we upsert or update the list
         updateDownloadInList(queryClient, data);
      }
      // Also invalidate to be sure
      queryClient.invalidateQueries({ queryKey: ['gravity', 'downloads'] });
      // And stats
      queryClient.invalidateQueries({ queryKey: ['gravity', 'stats'] });
      // And the specific task details
      if (data.id) {
        queryClient.invalidateQueries({ queryKey: ['gravity', 'status', data.id] });
      }
      break;
  }
}

function updateDownloadProgress(queryClient: any, progressData: any) {
  // Update List
  queryClient.setQueriesData({ queryKey: ['gravity', 'downloads'] }, (oldData: any) => {
    if (!oldData) return oldData;
    // oldData is Download[] because fetchDownloads returns res.data

    return oldData.map((d: Download) => {
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
    });
  });

  // Update Detail Page
  queryClient.setQueryData(['gravity', 'status', progressData.id], (oldData: any) => {
    if (!oldData) return oldData;
    
    return {
      ...oldData,
      downloaded: progressData.downloaded,
      size: progressData.size,
      speed: progressData.speed,
      eta: progressData.eta,
      seeders: progressData.seeders,
      peers: progressData.peers,
    };
  });
}

function updateUploadProgress(queryClient: any, progressData: any) {
  // Update List
  queryClient.setQueriesData({ queryKey: ['gravity', 'downloads'] }, (oldData: any) => {
    if (!oldData) return oldData;

    return oldData.map((d: Download) => {
      if (d.id === progressData.id) {
        return {
          ...d,
          uploadProgress: typeof progressData.uploaded === 'number' && progressData.size > 0
            ? Math.floor((progressData.uploaded / progressData.size) * 100)
            : d.uploadProgress,
          uploadSpeed: progressData.speed,
          // We might receive 'uploaded' bytes too if we want to track that
        };
      }
      return d;
    });
  });

  // Update Detail Page
  queryClient.setQueryData(['gravity', 'status', progressData.id], (oldData: any) => {
    if (!oldData) return oldData;

    return {
      ...oldData,
      uploadProgress: typeof progressData.uploaded === 'number' && progressData.size > 0
        ? Math.floor((progressData.uploaded / progressData.size) * 100)
        : oldData.uploadProgress,
      uploadSpeed: progressData.speed,
    };
  });
}

function updateStats(queryClient: any, statsData: any) {
  queryClient.setQueryData(['gravity', 'stats'], statsData);
}

function updateDownloadInList(queryClient: any, downloadData: any) {
    // If the event data is a full download object, we can update it in the list
    if (!downloadData || !downloadData.id) return;

    queryClient.setQueriesData({ queryKey: ['gravity', 'downloads'] }, (oldData: any) => {
        if (!oldData) return oldData;

        // Check if exists
        const exists = oldData.find((d: Download) => d.id === downloadData.id);

        if (exists) {
            return oldData.map((d: Download) =>
                d.id === downloadData.id ? { ...d, ...downloadData } : d
            );
        } else {
            // New download? 'download.created'
            // Only prepend if we have a full object roughly and it matches the list's implied filter?
            // Since we are updating ALL lists, blindly prepending might put an 'active' download in 'waiting' list cache.
            // Ideally we check if downloadData.status matches the query key, but query key is in the closure of setQueriesData iteration...
            // actually setQueriesData updater receives (oldData). It doesn't easily give access to the key being updated unless we use the functional form of setQueriesData(filter, updater).
            // For now, let's strictly UPDATE existing. For new items, we rely on invalidation (which happens right after this function in handleEvent).
            return oldData;
        }
    });

    // Update Detail Page if it exists
    queryClient.setQueryData(['gravity', 'status', downloadData.id], (oldData: any) => {
        if (!oldData) return oldData;
        return { ...oldData, ...downloadData };
    });
}
