import { useState } from 'react';

import { ApiError } from '@/shared/api/httpClient';
import { downloadReceiptPdf, reissueReceipt } from '@/features/receipts/api/receiptApi';
import { useReceipts } from '@/features/receipts/hooks/useReceipts';
import { TAX_CATEGORY_LABELS } from '@/features/receipts/types';
import type { Receipt } from '@/features/receipts/types';

const yen = (n: number) => n.toLocaleString('ja-JP');

/** 領収書の一覧・検索・PDF ダウンロード・再発行。 */
export function ReceiptList({ reloadKey }: { reloadKey?: number }) {
  const { receipts, loading, error, filter, setFilter, reload } = useReceipts();
  const [recipientQuery, setRecipientQuery] = useState('');
  const [actionError, setActionError] = useState<string | null>(null);

  // 親（作成成功）からの reloadKey 変化で再取得する。
  const [lastKey, setLastKey] = useState(reloadKey);
  if (reloadKey !== lastKey) {
    setLastKey(reloadKey);
    void reload();
  }

  const search = (e: React.FormEvent) => {
    e.preventDefault();
    setFilter({ ...filter, recipient: recipientQuery || undefined });
  };

  const download = async (r: Receipt) => {
    setActionError(null);
    try {
      const blob = await downloadReceiptPdf(r.id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `receipt-${r.id}.pdf`;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      setActionError(err instanceof ApiError ? `${err.status}: ${err.message}` : String(err));
    }
  };

  const reissue = async (r: Receipt) => {
    setActionError(null);
    try {
      await reissueReceipt(r.id);
      await download(r);
    } catch (err) {
      setActionError(err instanceof ApiError ? `${err.status}: ${err.message}` : String(err));
    }
  };

  return (
    <div style={{ display: 'grid', gap: '0.6rem' }}>
      <h3 style={{ margin: 0 }}>発行済みの領収書</h3>

      <form onSubmit={search} style={{ display: 'flex', gap: '0.4rem' }}>
        <input
          value={recipientQuery}
          onChange={(e) => setRecipientQuery(e.target.value)}
          placeholder="宛名で検索"
        />
        <button type="submit">検索</button>
      </form>

      {loading && <p>読み込み中…</p>}
      {error && <p style={{ color: 'crimson' }}>{error}</p>}
      {actionError && <p style={{ color: 'crimson' }}>{actionError}</p>}
      {!loading && !error && receipts.length === 0 && <p>領収書がありません。</p>}

      {receipts.map((r) => (
        <div
          key={r.id}
          style={{
            border: '1px solid #ddd',
            borderRadius: 6,
            padding: '0.6rem',
            display: 'flex',
            alignItems: 'center',
            gap: '0.75rem',
            flexWrap: 'wrap',
          }}
        >
          <strong>{r.recipient} 様</strong>
          <span>¥{yen(r.amount)}</span>
          <span style={{ color: '#666', fontSize: '0.85rem' }}>
            {TAX_CATEGORY_LABELS[r.tax_category]}（税 ¥{yen(r.tax_amount)}）
          </span>
          <span style={{ color: '#666', fontSize: '0.85rem' }}>{r.issued_on}</span>
          {r.proviso && <span style={{ color: '#888', fontSize: '0.8rem' }}>但し {r.proviso}</span>}
          <span style={{ marginLeft: 'auto', display: 'flex', gap: '0.4rem' }}>
            <button type="button" onClick={() => void download(r)}>
              PDF
            </button>
            <button type="button" onClick={() => void reissue(r)}>
              再発行
            </button>
          </span>
        </div>
      ))}
    </div>
  );
}
