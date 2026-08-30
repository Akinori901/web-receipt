import { useEffect, useState } from 'react';

import {
  getTenant,
  inviteMember,
  listMembers,
  updateTenant,
} from '@/features/organization/api/organizationApi';
import type { Member, Tenant, UpdateTenantInput } from '@/features/organization/types';
import { ApiError } from '@/shared/api/httpClient';

const EMPTY: UpdateTenantInput = {
  corporate_name: '',
  address: '',
  phone: '',
  email: '',
  registration_number: '',
};

const toForm = (t: Tenant): UpdateTenantInput => ({
  corporate_name: t.corporate_name,
  address: t.address,
  phone: t.phone,
  email: t.email,
  registration_number: t.registration_number,
});

const msg = (e: unknown) => (e instanceof ApiError ? `${e.status}: ${e.message}` : String(e));

/**
 * 組織パネル。発行者プロフィール（PDF に印字される情報）の編集と、
 * member の招待・一覧を扱う。admin のみが編集・招待でき、member は閲覧のみ。
 */
export function OrganizationPanel({ isAdmin }: { isAdmin: boolean }) {
  const [form, setForm] = useState<UpdateTenantInput>(EMPTY);
  const [members, setMembers] = useState<Member[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<string | null>(null);

  const set = <K extends keyof UpdateTenantInput>(k: K, v: UpdateTenantInput[K]) =>
    setForm((prev) => ({ ...prev, [k]: v }));

  useEffect(() => {
    let alive = true;
    void (async () => {
      setLoading(true);
      setError(null);
      try {
        const [t, m] = await Promise.all([getTenant(), listMembers()]);
        if (!alive) return;
        setForm(toForm(t));
        setMembers(m);
      } catch (e) {
        if (alive) setError(msg(e));
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, []);

  const saveProfile = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setSavedAt(null);
    try {
      const t = await updateTenant(form);
      setForm(toForm(t));
      setSavedAt(new Date().toLocaleTimeString('ja-JP'));
    } catch (err) {
      setError(msg(err));
    }
  };

  return (
    <section className="stack">
      <ProfileCard
        isAdmin={isAdmin}
        form={form}
        loading={loading}
        error={error}
        savedAt={savedAt}
        onChange={set}
        onSubmit={saveProfile}
      />
      <MembersCard
        isAdmin={isAdmin}
        members={members}
        onInvited={(m) => setMembers((prev) => [...prev, m])}
        onError={setError}
      />
    </section>
  );
}

function ProfileCard({
  isAdmin,
  form,
  loading,
  error,
  savedAt,
  onChange,
  onSubmit,
}: {
  isAdmin: boolean;
  form: UpdateTenantInput;
  loading: boolean;
  error: string | null;
  savedAt: string | null;
  onChange: <K extends keyof UpdateTenantInput>(k: K, v: UpdateTenantInput[K]) => void;
  onSubmit: (e: React.FormEvent) => void;
}) {
  return (
    <div className="card stack">
      <div className="row">
        <h2>発行者情報</h2>
        <span className="spacer" />
        {!isAdmin && <span className="badge badge-muted">閲覧のみ</span>}
      </div>
      <p className="muted" style={{ margin: 0, fontSize: '0.85rem' }}>
        ここに設定した内容が領収書 PDF の発行者欄・押印に印字されます。
      </p>

      {loading ? (
        <p className="muted">読み込み中…</p>
      ) : (
        <form className="stack-sm" onSubmit={onSubmit}>
          <label className="field">
            <span>法人名（または氏名）</span>
            <input
              className="input"
              value={form.corporate_name}
              onChange={(e) => onChange('corporate_name', e.target.value)}
              disabled={!isAdmin}
              placeholder="株式会社サンプル"
            />
          </label>
          <label className="field">
            <span>住所</span>
            <input
              className="input"
              value={form.address}
              onChange={(e) => onChange('address', e.target.value)}
              disabled={!isAdmin}
              placeholder="東京都千代田区…"
            />
          </label>
          <div className="row" style={{ gap: '0.75rem' }}>
            <label className="field" style={{ flex: 1, minWidth: 180 }}>
              <span>代表電話番号</span>
              <input
                className="input"
                value={form.phone}
                onChange={(e) => onChange('phone', e.target.value)}
                disabled={!isAdmin}
                placeholder="03-1234-5678"
              />
            </label>
            <label className="field" style={{ flex: 1, minWidth: 180 }}>
              <span>代表メール</span>
              <input
                className="input"
                type="email"
                value={form.email}
                onChange={(e) => onChange('email', e.target.value)}
                disabled={!isAdmin}
                placeholder="info@example.com"
              />
            </label>
          </div>
          <label className="field">
            <span>インボイス登録番号（T+13桁）</span>
            <input
              className="input input-mono"
              value={form.registration_number}
              onChange={(e) => onChange('registration_number', e.target.value)}
              disabled={!isAdmin}
              placeholder="T1234567890123"
            />
          </label>

          {error && <p className="error-text">{error}</p>}

          {isAdmin && (
            <div className="row">
              <button type="submit" className="btn btn-primary">
                保存する
              </button>
              {savedAt && <span className="badge badge-success">保存しました {savedAt}</span>}
            </div>
          )}
        </form>
      )}
    </div>
  );
}

function MembersCard({
  isAdmin,
  members,
  onInvited,
  onError,
}: {
  isAdmin: boolean;
  members: Member[];
  onInvited: (m: Member) => void;
  onError: (msg: string) => void;
}) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    onError('');
    try {
      const m = await inviteMember({ email, password });
      onInvited(m);
      setEmail('');
      setPassword('');
    } catch (err) {
      onError(msg(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="card stack">
      <h2>メンバー</h2>
      <p className="muted" style={{ margin: 0, fontSize: '0.85rem' }}>
        同じ組織のメンバーは、互いに発行した領収書をすべて履歴で参照できます。
      </p>

      <div className="stack-sm">
        {members.length === 0 && <p className="muted">メンバーがいません。</p>}
        {members.map((m) => (
          <div key={m.user_id} className="row" style={{ gap: '0.6rem' }}>
            <span>{m.email}</span>
            <span
              className={`badge ${m.role === 'admin' ? 'badge-accent' : 'badge-muted'}`}
            >
              {m.role === 'admin' ? '管理者' : 'メンバー'}
            </span>
            {!m.is_active && <span className="badge badge-danger">無効</span>}
          </div>
        ))}
      </div>

      {isAdmin && (
        <form className="stack-sm" onSubmit={submit} style={{ marginTop: '0.5rem' }}>
          <h3>メンバーを招待</h3>
          <div className="row" style={{ gap: '0.6rem' }}>
            <input
              className="input"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="member@example.com"
              required
              style={{ flex: 1, minWidth: 200 }}
            />
            <input
              className="input"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="初期パスワード(8文字以上)"
              required
              style={{ flex: 1, minWidth: 200 }}
            />
            <button type="submit" className="btn btn-primary" disabled={busy || password.length < 8}>
              {busy ? '登録中…' : '招待する'}
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
