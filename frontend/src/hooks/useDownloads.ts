import {
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { toast } from 'sonner';
import { api } from '../lib/api';
import { useEffect } from 'react';
import { useWebSocket } from './useWebSocket';

export function useDownloads(params?: { status?: string[]; limit?: number; offset?: number }) {
  const queryClient = useQueryClient();
  const { lastEvent } = useWebSocket();

  const query = useQuery({
    queryKey: ['downloads', params],
    queryFn: () => api.getDownloads(params),
    refetchInterval: 5000, // Background fallback
  });

  // Handle real-time updates
  useEffect(() => {
    if (!lastEvent) return;

    if (lastEvent.type.startsWith('download.')) {
      // For progress updates, we could update the cache directly for ultra-smooth UI
      // but for now, let's just invalidate to keep it simple and consistent
      queryClient.invalidateQueries({ queryKey: ['downloads'] });
    }
    
    if (lastEvent.type === 'download.completed') {
      toast.success(`Download complete: ${lastEvent.data.filename}`);
    }

    if (lastEvent.type === 'download.error') {
      toast.error(`Download failed: ${lastEvent.data.error}`);
    }
  }, [lastEvent, queryClient]);

  return query;
}

export function useDownloadActions() {
  const queryClient = useQueryClient();

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['downloads'] });
    queryClient.invalidateQueries({ queryKey: ['stats'] });
  };

  const create = useMutation({
    mutationFn: ({ url, destination, filename }: { url: string; destination?: string; filename?: string }) =>
      api.createDownload(url, destination, filename),
    onSuccess: () => {
      invalidate();
      toast.success('Download started');
    },
    onError: (err: Error) => toast.error(`Failed to start download: ${err.message}`),
  });

  const pause = useMutation({
    mutationFn: (id: string) => api.pauseDownload(id),
    onSuccess: () => {
      invalidate();
      toast.info('Download paused');
    },
  });

  const resume = useMutation({
    mutationFn: (id: string) => api.resumeDownload(id),
    onSuccess: () => {
      invalidate();
      toast.success('Download resumed');
    },
  });

  const remove = useMutation({
    mutationFn: ({ id, deleteFiles }: { id: string; deleteFiles?: boolean }) => api.deleteDownload(id, deleteFiles),
    onSuccess: () => {
      invalidate();
      toast.info('Download removed');
    },
  });

  return { create, pause, resume, remove };
}
