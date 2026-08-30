'use client';

import { useCallback, useEffect, useState } from 'react';
import { api, ApiError } from '@/lib/api';

export type ResourceState<T> =
  | { status: 'loading'; data: null; error: null }
  | { status: 'ok'; data: T; error: null }
  | { status: 'error'; data: null; error: ApiError | Error };

/**
 * GET нэг удаа — ачаалалт / өгөгдөл / алдааны гурван төлөвийг ил гаргана.
 * Хуудас бүр эдгээрийг зурах үүрэгтэй (чимээгүй хоосон хүснэгт үлдээхгүй).
 * 401-ийг api клиент өөрөө /login руу шилжүүлдэг тул энд алдаа болгохгүй.
 */
export function useResource<T>(path: string): ResourceState<T> & { reload: () => void } {
  const [state, setState] = useState<ResourceState<T>>({
    status: 'loading',
    data: null,
    error: null,
  });
  const [nonce, setNonce] = useState(0);

  useEffect(() => {
    let alive = true;
    setState({ status: 'loading', data: null, error: null });
    api
      .get<T>(path)
      .then((data) => alive && setState({ status: 'ok', data, error: null }))
      .catch((e: unknown) => {
        if (!alive) return;
        if (e instanceof ApiError && e.status === 401) return;
        setState({
          status: 'error',
          data: null,
          error: e instanceof Error ? e : new Error(String(e)),
        });
      });
    return () => {
      alive = false;
    };
  }, [path, nonce]);

  const reload = useCallback(() => setNonce((n) => n + 1), []);
  return { ...state, reload };
}
