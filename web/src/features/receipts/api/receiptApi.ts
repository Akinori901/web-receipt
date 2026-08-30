import { request, requestBlob } from '@/shared/api/httpClient';
import type { CreateReceiptInput, Receipt, ReceiptFilter } from '@/features/receipts/types';

// 認証トークンの付与・エラー処理は httpClient が担う。ここは薄い呼び出しだけ。

export function listReceipts(filter: ReceiptFilter = {}): Promise<Receipt[]> {
  const q = new URLSearchParams();
  if (filter.recipient) q.set('recipient', filter.recipient);
  if (filter.issued_from) q.set('issued_from', filter.issued_from);
  if (filter.issued_to) q.set('issued_to', filter.issued_to);
  const suffix = q.toString() ? `?${q.toString()}` : '';
  return request<Receipt[]>(`/api/receipts${suffix}`);
}

export function createReceipt(input: CreateReceiptInput): Promise<Receipt> {
  return request<Receipt>('/api/receipts', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function getReceipt(id: string): Promise<Receipt> {
  return request<Receipt>(`/api/receipts/${id}`);
}

export function reissueReceipt(id: string): Promise<Receipt> {
  return request<Receipt>(`/api/receipts/${id}/reissue`, { method: 'POST' });
}

export function downloadReceiptPdf(id: string): Promise<Blob> {
  return requestBlob(`/api/receipts/${id}/pdf`);
}
