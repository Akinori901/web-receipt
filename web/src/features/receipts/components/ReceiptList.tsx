import { useState, type ReactNode } from 'react';

import { ApiError } from '@/shared/api/httpClient';
import { downloadReceiptPdf, reissueReceipt } from '@/features/receipts/api/receiptApi';
import { useReceipts } from '@/features/receipts/hooks/useReceipts';
import { TAX_CATEGORY_LABELS } from '@/features/receipts/types';
import type { Receipt } from '@/features/receipts/types';

const yen = (n: number) => n.toLocaleString('ja-JP');

/**
 * 領収書の一覧・検索・PDF ダウンロード・再発行。
 *
 * 受け渡し（handover）は別 feature なので、feature 間 import を避けるため
 * renderHandover の render-prop で各行に差し込む（合成は app 層が行う）。
 */
export function ReceiptList({
  reloadKey,
  renderHandover,
}: {
  reloadKey?: number;
  renderHandover?: (receiptId: string) => ReactNode;
}) {
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
    <div className="card stack">
      <div className="row">
        <h2>発行済みの領収書</h2>
        <span className="spacer" />
        <form className="row" onSubmit={search} style={{ gap: '0.4rem' }}>
          <input
            className="input"
            value={recipientQuery}
            onChange={(e) => setRecipientQuery(e.target.value)}
            placeholder="宛名で検索"
            style={{ width: 180 }}
          />
          <button type="submit" className="btn btn-sm">
            検索
          </button>
        </form>
      </div>

      {loading && <p className="muted">読み込み中…</p>}
      {error && <p className="error-text">{error}</p>}
      {actionError && <p className="error-text">{actionError}</p>}
      {!loading && !error && receipts.length === 0 && <p className="muted">領収書がありません。</p>}

      <div className="stack-sm">
        {receipts.map((r) => (
          <div key={r.id} className="card" style={{ boxShadow: 'none', background: 'var(--surface-2)' }}>
            <div className="row" style={{ gap: '0.75rem' }}>
              <strong>{r.recipient} 様</strong>
              <span style={{ fontWeight: 700 }}>¥{yen(r.amount)}</span>
              <span className="badge badge-muted">
                {TAX_CATEGORY_LABELS[r.tax_category]}（税 ¥{yen(r.tax_amount)}）
              </span>
              <span className="muted" style={{ fontSize: '0.85rem' }}>
                {r.issued_on}
              </span>
              {r.proviso && (
                <span className="muted" style={{ fontSize: '0.8rem' }}>
                  但し {r.proviso}
                </span>
              )}
              <span className="spacer" />
              <button type="button" className="btn btn-sm" onClick={() => void download(r)}>
                PDF
              </button>
              <button type="button" className="btn btn-sm" onClick={() => void reissue(r)}>
                再発行
              </button>
            </div>
            {renderHandover?.(r.id)}
          </div>
        ))}
      </div>
    </div>
  );
}
