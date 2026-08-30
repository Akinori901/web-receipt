import { request } from '@/shared/api/httpClient';
import type { HandoverStatus } from '@/features/handover/types';

// 発行側（認証）の受け渡しAPI。これらの経路はアクセス数を加算しない
// （加算は受領側の非認証経路 /api/handover/... のみ）。

/** 領収書に閲覧ID/PWを発行する。既に発行済みなら既存の状況を返す。 */
export function issueHandover(receiptId: string): Promise<HandoverStatus> {
  return request<HandoverStatus>(`/api/receipts/${receiptId}/handover`, { method: 'POST' });
}

/** 発行済みの受け渡し状況（閲覧数・PDF出力数・宛名確定）を取得する。 */
export function getHandoverStatus(receiptId: string): Promise<HandoverStatus> {
  return request<HandoverStatus>(`/api/receipts/${receiptId}/handover`);
}
