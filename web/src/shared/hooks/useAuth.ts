import { useCallback, useState } from 'react';

import { fetchCurrentUser, signIn as signInApi } from '@/shared/api/authApi';
import { authStore } from '@/shared/api/authStore';

type Status = 'anonymous' | 'authenticated';

/**
 * 認証状態フック。
 *
 * バックエンドの /auth/sign-in（email + password）でトークンを取得し、
 * authStore にメモリ保持する。role（admin/member）も保持し、admin 判定に使う。
 * バックエンドが Cognito の InitiateAuth を呼ぶため、ローカルの cognito-local でも
 * 本番の実 Cognito でも同じコードパスで動く。
 *
 * アクセストークンは localStorage に置かず、リロードで消える（メモリ保持）。
 */
export function useAuth() {
  const [status, setStatus] = useState<Status>('anonymous');
  const [email, setEmail] = useState('');
  const [role, setRole] = useState('');

  const signIn = useCallback(async (inputEmail: string, password: string) => {
    const res = await signInApi(inputEmail, password);
    authStore.set(res.access_token);
    setEmail(res.user.email);
    setRole(res.user.role);
    setStatus('authenticated');
    try {
      const me = await fetchCurrentUser();
      setEmail(me.email);
      setRole(me.role);
    } catch {
      // /auth/me が失敗してもサインイン自体は成立している。
    }
  }, []);

  const signOut = useCallback(() => {
    authStore.set(null);
    setEmail('');
    setRole('');
    setStatus('anonymous');
  }, []);

  return { status, email, role, isAdmin: role === 'admin', signIn, signOut };
}
