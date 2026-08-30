// 受け渡し(handover) feature の型。バックエンドの handoverStatusResponse に対応する。
// 発行側（認証）が扱うアクセス状況を表す。

export type HandoverStatus = {
  view_id: string;
  view_password: string;
  url_view_count: number;
  pdf_count: number;
  recipient_confirmed: boolean;
};
