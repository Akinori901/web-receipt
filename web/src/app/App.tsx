import { useState } from 'react';
import { BrowserRouter, Route, Routes, useParams } from 'react-router';

import { SignInPanel } from '@/features/auth/components/SignInPanel';
import { HandoverPanel } from '@/features/handover/components/HandoverPanel';
import { OrganizationPanel } from '@/features/organization/components/OrganizationPanel';
import { ReceiptForm } from '@/features/receipts/components/ReceiptForm';
import { ReceiptList } from '@/features/receipts/components/ReceiptList';
import { ReceiptViewPage } from '@/features/receipt-view/components/ReceiptViewPage';
import { useAuth } from '@/shared/hooks/useAuth';

/**
 * app は組立点。ルーティングと feature の合成のみを行い、ロジックを持たない。
 *
 * ルートは2つ:
 *  - `/`            発行側（認証）: 領収書の発行・一覧・受け渡し・組織管理
 *  - `/view/:viewID` 受領側（非認証・別経路）: 閲覧・PDF出力・宛名確定
 *
 * 受領側を別ルート＝別経路にすることで、発行側の閲覧が
 * アクセスカウントに影響しない（カウント分離はバックエンド側でも担保）。
 */
export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<IssuerApp />} />
        <Route path="/view/:viewID" element={<ReceiverRoute />} />
      </Routes>
    </BrowserRouter>
  );
}

/** 発行側アプリ（認証必須）。 */
function IssuerApp() {
  // 認証状態は app 層で一元管理し、各 feature に props で渡す。
  const { status, email, role, isAdmin, signIn, signOut } = useAuth();
  const [tab, setTab] = useState<'receipts' | 'organization'>('receipts');
  const [reloadKey, setReloadKey] = useState(0);

  if (status !== 'authenticated') {
    return (
      <>
        <header className="appbar">
          <div className="container" style={{ padding: 0 }}>
            <strong>Web 領収書</strong>
          </div>
        </header>
        <SignInPanel status={status} email={email} role={role} onSignIn={signIn} onSignOut={signOut} />
      </>
    );
  }

  return (
    <>
      <header className="appbar">
        <div className="container row" style={{ padding: 0 }}>
          <strong>Web 領収書</strong>
          <span className="spacer" />
          <SignInPanel status={status} email={email} role={role} onSignIn={signIn} onSignOut={signOut} />
        </div>
      </header>

      <div className="container stack">
        <nav className="tabs">
          <button
            type="button"
            className={`tab ${tab === 'receipts' ? 'tab-active' : ''}`}
            onClick={() => setTab('receipts')}
          >
            領収書
          </button>
          <button
            type="button"
            className={`tab ${tab === 'organization' ? 'tab-active' : ''}`}
            onClick={() => setTab('organization')}
          >
            組織
          </button>
        </nav>

        {tab === 'receipts' ? (
          <div className="stack">
            <ReceiptForm onCreated={() => setReloadKey((k) => k + 1)} />
            <ReceiptList
              reloadKey={reloadKey}
              renderHandover={(receiptId) => <HandoverPanel receiptId={receiptId} />}
            />
          </div>
        ) : (
          <OrganizationPanel isAdmin={isAdmin} />
        )}
      </div>
    </>
  );
}

/** 受領側ルート（非認証）。URL の :viewID を受領側ページへ渡す。 */
function ReceiverRoute() {
  const { viewID } = useParams();
  if (!viewID) return null;
  return (
    <div style={{ paddingTop: '2rem' }}>
      <ReceiptViewPage viewID={viewID} />
    </div>
  );
}
