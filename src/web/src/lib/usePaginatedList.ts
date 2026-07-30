import { useState, useEffect, useCallback, useRef } from 'react';
import api from '@/lib/api';

export interface Message {
  id: string;
  direction: string;
  phone_number: string;
  body: string;
  status: string;
  created_at: string;
}

export const PAGE_SIZE_OPTIONS = [50, 100, 200] as const;
export const DEFAULT_PAGE_SIZE = 50;

const PAGE_SIZE_STORAGE_KEY = 'sms-gateway.pageSize';

function loadPageSize(): number {
  const stored = Number(localStorage.getItem(PAGE_SIZE_STORAGE_KEY));
  return PAGE_SIZE_OPTIONS.includes(stored as (typeof PAGE_SIZE_OPTIONS)[number])
    ? stored
    : DEFAULT_PAGE_SIZE;
}

interface UsePaginatedListResult<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
  /** True only on the first load, when there is nothing to show yet. */
  loading: boolean;
  /** True while fetching a page with previous results still on screen. */
  refreshing: boolean;
  error: string;
  setPage: (page: number) => void;
  setPageSize: (size: number) => void;
  /** Drops ids locally and refetches, so a deletion cannot leave a short page. */
  removeItems: (ids: Set<string>) => void;
  /** Refetches the current page, e.g. after creating a record. */
  refresh: () => void;
}

/**
 * Fetches one page of a listing at a time.
 *
 * The API returns a bare array with the unpaginated total in X-Total-Count, so
 * this reads the header rather than a response envelope. That shape is what
 * keeps existing API-key clients working, and it means `total` is the count of
 * everything matching the filter — not the length of the page.
 *
 * Items only need an `id`; messages, users and API keys all qualify.
 */
export function usePaginatedList<T extends { id: string }>(
  path: string,
  params?: Record<string, string>,
): UsePaginatedListResult<T> {
  const [items, setItems] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSizeState] = useState(loadPageSize);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');
  const [reloadToken, setReloadToken] = useState(0);

  // Serializing here keeps `params` out of the effect's dependency list, so a
  // caller passing an inline object literal does not retrigger every render.
  const paramsKey = JSON.stringify(params ?? {});
  const hasLoadedOnce = useRef(false);

  useEffect(() => {
    let cancelled = false;

    const fetchPage = async () => {
      if (hasLoadedOnce.current) setRefreshing(true);

      try {
        const res = await api.get<T[]>(path, {
          params: {
            ...(JSON.parse(paramsKey) as Record<string, string>),
            limit: pageSize,
            offset: (page - 1) * pageSize,
          },
        });
        if (cancelled) return;

        setItems(res.data);
        // Absent header means an older server that ignores limit/offset; falling
        // back to the page length keeps the UI coherent instead of showing 0.
        const header = res.headers['x-total-count'];
        setTotal(header !== undefined ? Number(header) : res.data.length);
        setError('');
      } catch {
        if (!cancelled) setError('Failed to load messages.');
      } finally {
        if (!cancelled) {
          setLoading(false);
          setRefreshing(false);
          hasLoadedOnce.current = true;
        }
      }
    };

    fetchPage();
    return () => {
      cancelled = true;
    };
  }, [path, paramsKey, page, pageSize, reloadToken]);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  // Deleting the last messages on the final page would otherwise strand the user
  // on a page that no longer exists.
  useEffect(() => {
    if (page > totalPages) setPage(totalPages);
  }, [page, totalPages]);

  const setPageSize = useCallback((size: number) => {
    localStorage.setItem(PAGE_SIZE_STORAGE_KEY, String(size));
    setPageSizeState(size);
    // Offsets are meaningless across page sizes; go back to the first page.
    setPage(1);
  }, []);

  const refresh = useCallback(() => setReloadToken((t) => t + 1), []);

  const removeItems = useCallback((ids: Set<string>) => {
    // Drop them immediately so the UI responds, then refetch: the rows that
    // shift up from the next page can only come from the server.
    setItems((prev) => prev.filter((item) => !ids.has(item.id)));
    setTotal((prev) => Math.max(0, prev - ids.size));
    setReloadToken((t) => t + 1);
  }, []);

  return {
    items,
    total,
    page,
    pageSize,
    totalPages,
    loading,
    refreshing,
    error,
    setPage,
    setPageSize,
    removeItems,
    refresh,
  };
}
