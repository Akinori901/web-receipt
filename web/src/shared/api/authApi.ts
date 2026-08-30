import { request } from '@/shared/api/httpClient';

// 認証 API。useAuth（shared/hooks）から使うため shared に置く。
// httpClient を叩くだけの薄い層で、feature には依存しない。

export type CurrentUser = {
  id: string;
  email: string;
  display_name: string;
  is_active: boolean;
  tenant_id: string;
  role: string;
};

export type SignInResponse = {
  access_token: string;
  id_token: string;
  refresh_token: string;
  expires_in: number;
  user: {
    user_id: string;
    email: string;
    display_name: string;
    is_active: boolean;
    tenant_id: string;
    role: string;
  };
};

/**
 * バックエンドの /auth/sign-in を叩く（email + password）。
 *
 * バックエンドが Cognito の InitiateAuth を呼んでトークンを返す。
 * この方式なら、ローカルの cognito-local でも本番の実 Cognito でも
 * 同じコードパスで動く（OIDC のホストされたログイン画面に依存しない）。
 */
export function signIn(email: string, password: string): Promise<SignInResponse> {
  return request<SignInResponse>('/api/auth/sign-in', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

/** バックエンドの /auth/me を叩いて自サービス側のユーザー情報を取る。 */
export function fetchCurrentUser(): Promise<CurrentUser> {
  return request<CurrentUser>('/api/auth/me');
}
