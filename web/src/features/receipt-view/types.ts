// 受領側(receipt-view) feature の型。バックエンドの receiptViewResponse に対応する。
// この feature は非認証の別ルート /view/:viewID から使う。

export type ReceiptView = {
  recipient: string;
  amount: number;
  tax_category: string;
  issued_on: string;
  proviso: string;
  recipient_confirmed: boolean;
};

// バックエンドの tax_category 文字列を日本語ラベルへ。
export const TAX_LABELS: Record<string, string> = {
  standard10: '標準税率 10%',
  reduced8: '軽減税率 8%',
  exempt: '非課税・対象外',
};
