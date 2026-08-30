import { useState } from 'react';

import { ApiError } from '@/shared/api/httpClient';

type Status = 'anonymous' | 'authenticated';

/**
 * サインインパネル。認証状態は app 層で一元管理し、props で受け取る
 * （feature 間・hook の多重呼び出しで状態がずれるのを避ける）。
 */
export function SignInPanel({
  status,
  email,
  role,
  onSignIn,
  onSignOut,
}: {
  status: Status;
  email: string;
  role: string;
  onSignIn: (email: string, password: string) => Promise<void>;
  onSignOut: () => void;
}) {
  const [inputEmail, setInputEmail] = useState('taro@example.com');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (status === 'authenticated') {
    return (
      <div className="row">
        <span className="badge badge-accent">{role === 'admin' ? '管理者' : 'メンバー'}</span>
        <span className="muted">{email}</span>
        <span className="spacer" />
        <button type="button" className="btn btn-sm" onClick={onSignOut}>
          サインアウト
        </button>
      </div>
    );
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await onSignIn(inputEmail, password);
    } catch (err) {
      setError(err instanceof ApiError ? `${err.status}: ${err.message}` : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="container-narrow">
      <div className="card card-pad-lg stack">
        <div className="stack-sm">
          <h1>Web 領収書</h1>
          <p className="muted" style={{ margin: 0 }}>
            サインインして領収書を発行・管理します。
          </p>
        </div>
        <form className="stack-sm" onSubmit={submit}>
          <label className="field">
            <span>メールアドレス</span>
            <input
              className="input"
              type="email"
              value={inputEmail}
              onChange={(e) => setInputEmail(e.target.value)}
              placeholder="you@example.com"
              required
            />
          </label>
          <label className="field">
            <span>パスワード</span>
            <input
              className="input"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="パスワード"
              required
            />
          </label>
          {error && <p className="error-text">{error}</p>}
          <button type="submit" className="btn btn-primary" disabled={busy || !password}>
            {busy ? 'サインイン中…' : 'サインイン'}
          </button>
        </form>
      </div>
    </div>
  );
}
