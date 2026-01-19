import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import { toast } from 'sonner';

export function useProviders() {
  return useQuery({
    queryKey: ['providers'],
    queryFn: () => api.getProviders(),
  });
}

export function useProviderActions() {
  const queryClient = useQueryClient();

  const configure = useMutation({
    mutationFn: ({ name, config, enabled }: { name: string; config: Record<string, string>; enabled: boolean }) =>
      api.configureProvider(name, config, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['providers'] });
      toast.success('Provider configured');
    },
    onError: (err: Error) => toast.error(`Failed to configure provider: ${err.message}`),
  });

  return { configure };
}
