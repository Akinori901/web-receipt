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
  onSignIn,
  onSignOut,
}: {
  status: Status;
  email: string;
  onSignIn: (email: string, password: string) => Promise<void>;
  onSignOut: () => void;
}) {
  const [inputEmail, setInputEmail] = useState('taro@example.com');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (status === 'authenticated') {
    return (
      <section>
        <span>サインイン済み: {email}</span>{' '}
        <button onClick={onSignOut}>サインアウト</button>
      </section>
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
    <section>
      <h2>サインイン</h2>
      <form onSubmit={submit} style={{ display: 'grid', gap: '0.5rem', maxWidth: 320 }}>
        <input
          type="email"
          value={inputEmail}
          onChange={(e) => setInputEmail(e.target.value)}
          placeholder="メールアドレス"
          required
        />
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="パスワード"
          required
        />
        {error && <p style={{ color: 'crimson' }}>{error}</p>}
        <button type="submit" disabled={busy || !password}>
          {busy ? 'サインイン中…' : 'サインイン'}
        </button>
      </form>
    </section>
  );
}
