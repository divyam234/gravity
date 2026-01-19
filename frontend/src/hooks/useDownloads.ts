import {
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { toast } from 'sonner';
import { api } from '../lib/api';
import { useSettingsStore } from '../store/useSettingsStore';

export function useDownloads(params?: { status?: string[]; limit?: number; offset?: number }) {
  const { pollingInterval } = useSettingsStore();

  return useQuery({
    queryKey: ['downloads', params],
    queryFn: () => api.getDownloads(params),
    refetchInterval: (query) => (query.state.status === 'error' ? false : pollingInterval),
  });
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