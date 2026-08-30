import { useCallback, useState } from 'react';

import { fetchCurrentUser, signIn as signInApi } from '@/shared/api/authApi';
import { authStore } from '@/shared/api/authStore';

type Status = 'anonymous' | 'authenticated';

/**
 * 認証状態フック。
 *
 * バックエンドの /auth/sign-in（email + password）でトークンを取得し、
 * authStore にメモリ保持する。バックエンドが Cognito の InitiateAuth を
 * 呼ぶため、ローカルの cognito-local でも本番の実 Cognito でも同じコードパスで
 * 動く（OIDC のホストされたログイン画面に依存しない）。
 *
 * アクセストークンは localStorage に置かず、リロードで消える（メモリ保持）。
 */
export function useAuth() {
  const [status, setStatus] = useState<Status>('anonymous');
  const [email, setEmail] = useState('');
  const [error, setError] = useState<string | null>(null);

  const signIn = useCallback(async (inputEmail: string, password: string) => {
    setError(null);
    const res = await signInApi(inputEmail, password);
    authStore.set(res.access_token);
    setEmail(res.user.email);
    setStatus('authenticated');
    // 自サービス側のユーザー情報も取得しておく（表示名等）。失敗しても致命ではない。
    try {
      const me = await fetchCurrentUser();
      setEmail(me.email);
    } catch {
      // /auth/me が失敗してもサインイン自体は成立している。
    }
  }, []);

  const signOut = useCallback(() => {
    authStore.set(null);
    setEmail('');
    setStatus('anonymous');
  }, []);

  return { status, email, error, setError, signIn, signOut };
}
