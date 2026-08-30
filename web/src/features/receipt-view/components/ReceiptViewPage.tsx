import { useState } from 'react';

import {
  confirmRecipient,
  outputReceiptPdf,
  viewReceipt,
} from '@/features/receipt-view/api/receiptViewApi';
import { TAX_LABELS, type ReceiptView } from '@/features/receipt-view/types';
import { ApiError } from '@/shared/api/httpClient';

const yen = (n: number) => n.toLocaleString('ja-JP');
const msg = (e: unknown) => (e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));

/**
 * 受領側ページ（非認証・別ルート /view/:viewID）。
 *
 * 発行側から連絡された閲覧パスワードを入力して領収書を確認し、
 * PDF を出力できる。宛名は1回だけ確定でき、確定後は変更不可。
 *
 * 認証を通さないため、ここでのアクセスはすべてカウント対象になる
 * （発行側の閲覧とはエンドポイントが分かれている）。
 */
export function ReceiptViewPage({ viewID }: { viewID: string }) {
  const [password, setPassword] = useState('');
  const [receipt, setReceipt] = useState<ReceiptView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const open = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      setReceipt(await viewReceipt(viewID, password));
    } catch (err) {
      setError(msg(err));
    } finally {
      setBusy(false);
    }
  };

  if (!receipt) {
    return (
      <div className="container-narrow">
        <div className="card card-pad-lg stack">
          <div className="stack-sm">
            <h1>領収書の受け取り</h1>
            <p className="muted" style={{ margin: 0 }}>
              発行者から連絡された閲覧パスワードを入力してください。
            </p>
          </div>
          <form className="stack-sm" onSubmit={open}>
            <label className="field">
              <span>閲覧パスワード</span>
              <input
                className="input input-mono"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoFocus
                required
              />
            </label>
            {error && <p className="error-text">{error}</p>}
            <button type="submit" className="btn btn-primary" disabled={busy || !password}>
              {busy ? '確認中…' : '領収書を表示'}
            </button>
          </form>
        </div>
      </div>
    );
  }

  return (
    <div className="container-narrow">
      <ReceiptCard viewID={viewID} password={password} receipt={receipt} onUpdate={setReceipt} />
    </div>
  );
}

function ReceiptCard({
  viewID,
  password,
  receipt,
  onUpdate,
}: {
  viewID: string;
  password: string;
  receipt: ReceiptView;
  onUpdate: (r: ReceiptView) => void;
}) {
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirmName, setConfirmName] = useState(receipt.recipient);

  const download = async () => {
    setBusy(true);
    setError(null);
    try {
      const blob = await outputReceiptPdf(viewID, password);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'receipt.pdf';
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(msg(err));
    } finally {
      setBusy(false);
    }
  };

  const confirm = async () => {
    setBusy(true);
    setError(null);
    try {
      await confirmRecipient(viewID, password, confirmName);
      onUpdate({ ...receipt, recipient: confirmName, recipient_confirmed: true });
    } catch (err) {
      setError(msg(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="card card-pad-lg stack">
      <div className="row">
        <h1>領収書</h1>
        <span className="spacer" />
        {receipt.recipient_confirmed ? (
          <span className="badge badge-success">宛名確定済み</span>
        ) : (
          <span className="badge badge-muted">宛名未確定</span>
        )}
      </div>

      <dl className="stack-sm" style={{ margin: 0 }}>
        <Row label="宛名" value={`${receipt.recipient} 様`} />
        <Row label="金額（税込）" value={`¥${yen(receipt.amount)}`} strong />
        <Row label="税区分" value={TAX_LABELS[receipt.tax_category] ?? receipt.tax_category} />
        <Row label="発行日" value={receipt.issued_on} />
        {receipt.proviso && <Row label="但し書き" value={receipt.proviso} />}
      </dl>

      {error && <p className="error-text">{error}</p>}

      <div className="row">
        <button type="button" className="btn btn-primary" onClick={() => void download()} disabled={busy}>
          {busy ? '処理中…' : 'PDFを出力'}
        </button>
      </div>

      {!receipt.recipient_confirmed && (
        <div className="card stack-sm" style={{ background: 'var(--surface-2)', boxShadow: 'none' }}>
          <h3>宛名を確定する</h3>
          <p className="muted" style={{ margin: 0, fontSize: '0.85rem' }}>
            確定は一度だけです。内容を確認してから確定してください。
          </p>
          <div className="row" style={{ gap: '0.4rem' }}>
            <input
              className="input"
              value={confirmName}
              onChange={(e) => setConfirmName(e.target.value)}
              style={{ flex: 1, minWidth: 200 }}
            />
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => void confirm()}
              disabled={busy || !confirmName.trim()}
            >
              確定する
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function Row({ label, value, strong }: { label: string; value: string; strong?: boolean }) {
  return (
    <div className="row" style={{ justifyContent: 'space-between', gap: '1rem' }}>
      <dt className="muted" style={{ fontSize: '0.85rem' }}>
        {label}
      </dt>
      <dd
        style={{ margin: 0, fontWeight: strong ? 700 : 500, fontSize: strong ? '1.1rem' : '1rem' }}
      >
        {value}
      </dd>
    </div>
  );
}
