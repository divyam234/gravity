import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { useWebSocket } from './useWebSocket';
import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';

export function useStats() {
  const queryClient = useQueryClient();
  const { lastEvent } = useWebSocket();

  const query = useQuery({
    queryKey: ['stats'],
    queryFn: () => api.getStats(),
    refetchInterval: (query) => (query.state.status === 'error' ? false : 5000),
  });

  useEffect(() => {
    if (lastEvent?.type === 'stats') {
      queryClient.setQueryData(['stats'], lastEvent.data);
    }
  }, [lastEvent, queryClient]);

  return query;
}
