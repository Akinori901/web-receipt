import { useCallback, useEffect, useState } from 'react';

import { ApiError } from '@/shared/api/httpClient';
import { listReceipts } from '@/features/receipts/api/receiptApi';
import type { Receipt, ReceiptFilter } from '@/features/receipts/types';

type State = {
  receipts: Receipt[];
  loading: boolean;
  error: string | null;
};

/**
 * 領収書一覧の取得と状態管理。filter を変えて reload できる。
 */
export function useReceipts(initialFilter: ReceiptFilter = {}) {
  const [state, setState] = useState<State>({ receipts: [], loading: false, error: null });
  const [filter, setFilter] = useState<ReceiptFilter>(initialFilter);

  const reload = useCallback(async () => {
    setState((s) => ({ ...s, loading: true, error: null }));
    try {
      const receipts = await listReceipts(filter);
      setState({ receipts, loading: false, error: null });
    } catch (e) {
      const message = e instanceof ApiError ? `${e.status}: ${e.message}` : String(e);
      setState({ receipts: [], loading: false, error: message });
    }
  }, [filter]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { ...state, filter, setFilter, reload };
}
