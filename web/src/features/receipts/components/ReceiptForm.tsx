import { useState } from 'react';

import { ApiError } from '@/shared/api/httpClient';
import { createReceipt } from '@/features/receipts/api/receiptApi';
import type { CreateReceiptInput, Receipt, TaxCategory } from '@/features/receipts/types';
import { TAX_CATEGORY_LABELS } from '@/features/receipts/types';

/** 領収書を発行するフォーム。成功したら onCreated に発行結果を渡す。 */
export function ReceiptForm({ onCreated }: { onCreated: (r: Receipt) => void }) {
  const [form, setForm] = useState<CreateReceiptInput>({
    recipient: '',
    recipient_address: '',
    amount: 0,
    tax_category: 'standard10',
    registration_number: '',
    proviso: '',
    issued_on: '',
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const set = <K extends keyof CreateReceiptInput>(k: K, v: CreateReceiptInput[K]) =>
    setForm((prev) => ({ ...prev, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const receipt = await createReceipt(form);
      onCreated(receipt);
      setForm({ ...form, recipient: '', recipient_address: '', amount: 0, registration_number: '', proviso: '' });
    } catch (err) {
      setError(err instanceof ApiError ? `${err.status}: ${err.message}` : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="card stack">
      <h2>領収書を発行</h2>
      <form className="stack-sm" onSubmit={submit}>
        <label className="field">
          <span>宛名（必須）</span>
          <input className="input" value={form.recipient} onChange={(e) => set('recipient', e.target.value)} required />
        </label>
        <label className="field">
          <span>宛先住所（任意）</span>
          <input
            className="input"
            value={form.recipient_address}
            onChange={(e) => set('recipient_address', e.target.value)}
          />
        </label>
        <div className="row" style={{ gap: '0.75rem' }}>
          <label className="field" style={{ flex: 1, minWidth: 160 }}>
            <span>金額（税込・円）</span>
            <input
              className="input"
              type="number"
              min={0}
              value={form.amount}
              onChange={(e) => set('amount', Number(e.target.value))}
              required
            />
          </label>
          <label className="field" style={{ flex: 1, minWidth: 160 }}>
            <span>税区分</span>
            <select
              className="select"
              value={form.tax_category}
              onChange={(e) => set('tax_category', e.target.value as TaxCategory)}
            >
              {(Object.keys(TAX_CATEGORY_LABELS) as TaxCategory[]).map((c) => (
                <option key={c} value={c}>
                  {TAX_CATEGORY_LABELS[c]}
                </option>
              ))}
            </select>
          </label>
        </div>
        <label className="field">
          <span>登録番号（インボイス T+13桁・任意）</span>
          <input
            className="input input-mono"
            value={form.registration_number}
            onChange={(e) => set('registration_number', e.target.value)}
            placeholder="T1234567890123"
          />
        </label>
        <label className="field">
          <span>但し書き（任意）</span>
          <input
            className="input"
            value={form.proviso}
            onChange={(e) => set('proviso', e.target.value)}
            placeholder="お品代として"
          />
        </label>
        <label className="field">
          <span>発行日（空なら当日）</span>
          <input className="input" type="date" value={form.issued_on} onChange={(e) => set('issued_on', e.target.value)} />
        </label>

        {error && <p className="error-text">{error}</p>}

        <button type="submit" className="btn btn-primary" disabled={busy || !form.recipient.trim()}>
          {busy ? '発行中…' : '発行する'}
        </button>
      </form>
    </div>
  );
}
