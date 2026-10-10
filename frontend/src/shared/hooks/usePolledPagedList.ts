import { type RefObject, useEffect, useState } from 'react';

import {
  PagedListController,
  type PagedListOptions,
  type PagedListState,
  createInitialPagedListState,
} from '../lib/PagedListController';

const SCROLL_LOAD_THRESHOLD_PX = 100;

interface Params<TItem, TCursor> extends PagedListOptions<TItem, TCursor> {
  refreshIntervalMs: number;
  scrollContainerRef?: RefObject<HTMLDivElement | null>;
  // A change discards the loaded rows and loads the first page again.
  resetKey: string;
}

// Pages a newest-first list as the user scrolls and keeps it current: each tick
// re-requests only the first page and merges it into the loaded rows.
export function usePolledPagedList<TItem, TCursor>({
  fetchPage,
  getItemKey,
  onError,
  refreshIntervalMs,
  scrollContainerRef,
  resetKey,
}: Params<TItem, TCursor>) {
  const [state, setState] = useState<PagedListState<TItem, TCursor>>(createInitialPagedListState);
  const [controller] = useState(
    () => new PagedListController<TItem, TCursor>({ fetchPage, getItemKey, onError }, setState),
  );

  // The interval and the scroll listener outlive a render, so the controller
  // reads the latest callbacks. Declared first so it runs before the reload.
  useEffect(() => {
    controller.setOptions({ fetchPage, getItemKey, onError });
  });

  useEffect(() => {
    controller.reload();
  }, [resetKey]);

  useEffect(() => {
    const intervalId = setInterval(() => controller.refreshFirstPage(), refreshIntervalMs);

    return () => clearInterval(intervalId);
  }, [refreshIntervalMs]);

  useEffect(() => {
    const container = scrollContainerRef?.current;
    if (!container) return;

    const handleScroll = () => {
      const { scrollTop, scrollHeight, clientHeight } = container;

      if (scrollHeight - scrollTop <= clientHeight + SCROLL_LOAD_THRESHOLD_PX) {
        controller.loadNextPage();
      }
    };

    container.addEventListener('scroll', handleScroll);
    return () => container.removeEventListener('scroll', handleScroll);
  }, [scrollContainerRef]);

  return {
    items: state.list.items,
    isLoading: state.isLoading,
    isLoadingMore: state.isLoadingMore,
    hasMore: state.list.nextCursor !== undefined,
    reload: () => controller.reload(),
    refreshFirstPage: () => controller.refreshFirstPage(),
  };
}
