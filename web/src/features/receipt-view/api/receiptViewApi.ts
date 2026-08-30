import { request, requestBlob } from '@/shared/api/httpClient';
import type { ReceiptView } from '@/features/receipt-view/types';

// 受領側（非認証）の受け渡しAPI。viewID(パス) + password(body) でアクセスする。
// httpClient は token が無ければ Authorization を付けないので、そのまま非認証で通る。
// これらの経路はアクセス数を加算する（発行側の閲覧とは経路が分かれている）。

/** 領収書を閲覧する（URL閲覧数 +1）。 */
export function viewReceipt(viewID: string, password: string): Promise<ReceiptView> {
  return request<ReceiptView>(`/api/handover/${viewID}/view`, {
    method: 'POST',
    body: JSON.stringify({ password }),
  });
}

/** 領収書 PDF を出力する（PDF出力数 +1）。 */
export function outputReceiptPdf(viewID: string, password: string): Promise<Blob> {
  return requestBlob(`/api/handover/${viewID}/pdf`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ password }),
  });
}

/** 宛名を確定する（1回のみ・2回目は 409）。 */
export function confirmRecipient(viewID: string, password: string, recipient: string): Promise<void> {
  return request<void>(`/api/handover/${viewID}/confirm`, {
    method: 'POST',
    body: JSON.stringify({ password, recipient }),
  });
}
