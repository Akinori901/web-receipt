import { useState } from 'react';

import { getHandoverStatus, issueHandover } from '@/features/handover/api/handoverApi';
import type { HandoverStatus } from '@/features/handover/types';
import { ApiError } from '@/shared/api/httpClient';

const msg = (e: unknown) => (e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));

// 受領側が使う閲覧 URL。同じアプリの非認証ルート /view/:viewID を指す。
const viewUrl = (viewID: string) => `${window.location.origin}/view/${viewID}`;

/**
 * 受け渡しパネル（発行側）。各領収書に対して閲覧ID/PWを発行し、
 * アクセス状況（URL閲覧数・PDF出力数・宛名確定）を表示する。
 *
 * ここは認証経路なので、状況の取得・発行でカウントは増えない。
 */
export function HandoverPanel({ receiptId }: { receiptId: string }) {
  const [status, setStatus] = useState<HandoverStatus | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const load = async () => {
    setBusy(true);
    setError(null);
    try {
      const s = await getHandoverStatus(receiptId);
      setStatus(s);
    } catch (e) {
      // 未発行なら 404。エラーにはせず「未発行」として扱う。
      if (!(e instanceof ApiError && e.status === 404)) setError(msg(e));
      setStatus(null);
    } finally {
      setLoaded(true);
      setBusy(false);
    }
  };

  const issue = async () => {
    setBusy(true);
    setError(null);
    try {
      setStatus(await issueHandover(receiptId));
    } catch (e) {
      setError(msg(e));
    } finally {
      setBusy(false);
    }
  };

  const copy = async (viewID: string) => {
    try {
      await navigator.clipboard.writeText(viewUrl(viewID));
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      // クリップボードが使えない環境では無視（URL は画面に表示済み）。
    }
  };

  if (!loaded) {
    return (
      <button type="button" className="btn btn-sm" onClick={() => void load()} disabled={busy}>
        {busy ? '確認中…' : '受け渡し状況'}
      </button>
    );
  }

  return (
    <div className="stack-sm" style={{ width: '100%', marginTop: '0.5rem' }}>
      {error && <p className="error-text">{error}</p>}

      {!status ? (
        <div className="row">
          <span className="badge badge-muted">未発行</span>
          <button type="button" className="btn btn-sm btn-primary" onClick={() => void issue()} disabled={busy}>
            {busy ? '発行中…' : '閲覧ID/PWを発行'}
          </button>
        </div>
      ) : (
        <div className="card stack-sm" style={{ background: 'var(--surface-2)', boxShadow: 'none' }}>
          <div className="row" style={{ gap: '0.5rem' }}>
            <span className="badge badge-accent">閲覧数 {status.url_view_count}</span>
            <span className="badge badge-accent">PDF出力 {status.pdf_count}</span>
            {status.recipient_confirmed ? (
              <span className="badge badge-success">宛名確定済み</span>
            ) : (
              <span className="badge badge-muted">宛名未確定</span>
            )}
          </div>
          <label className="field">
            <span>受領側URL（相手に連絡）</span>
            <div className="row" style={{ gap: '0.4rem' }}>
              <input className="input input-mono" readOnly value={viewUrl(status.view_id)} />
              <button type="button" className="btn btn-sm" onClick={() => void copy(status.view_id)}>
                {copied ? 'コピー済' : 'コピー'}
              </button>
            </div>
          </label>
          <label className="field">
            <span>閲覧パスワード</span>
            <input className="input input-mono" readOnly value={status.view_password} />
          </label>
        </div>
      )}
    </div>
  );
}
