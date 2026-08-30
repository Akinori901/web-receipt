import { useState } from 'react';

import { SignInPanel } from '@/features/auth/components/SignInPanel';
import { ReceiptForm } from '@/features/receipts/components/ReceiptForm';
import { ReceiptList } from '@/features/receipts/components/ReceiptList';
import { useAuth } from '@/shared/hooks/useAuth';

/**
 * app は組立点。feature を並べるだけで、ロジックを持たない。
 *
 * 領収書は「作成」と「一覧」の2ペインで足りるため、ルーティングは導入しない
 * （画面が増えたら react-router を app 層に足す）。
 */
export function App() {
  // 認証状態は app 層で一元管理し、SignInPanel に props で渡す。
  const { status, email, signIn, signOut } = useAuth();
  // 作成が成功するたびに +1 して一覧へ再取得を促す。
  const [reloadKey, setReloadKey] = useState(0);

  return (
    <main style={{ maxWidth: 960, margin: '0 auto', padding: '1.5rem' }}>
      <h1>Web 領収書</h1>
      <SignInPanel status={status} email={email} onSignIn={signIn} onSignOut={signOut} />

      {status === 'authenticated' && (
        <div style={{ display: 'grid', gap: '2rem', marginTop: '1.5rem' }}>
          <ReceiptForm onCreated={() => setReloadKey((k) => k + 1)} />
          <ReceiptList reloadKey={reloadKey} />
        </div>
      )}
    </main>
  );
}
