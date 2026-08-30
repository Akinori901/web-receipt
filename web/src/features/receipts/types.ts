// 領収書 feature の型。バックエンドの receiptResponse / createReceiptRequest に対応する。

export type TaxCategory = 'standard10' | 'reduced8' | 'exempt';

export const TAX_CATEGORY_LABELS: Record<TaxCategory, string> = {
  standard10: '標準税率 10%',
  reduced8: '軽減税率 8%',
  exempt: '非課税・対象外',
};

export type Receipt = {
  id: string;
  recipient: string;
  recipient_address: string;
  amount: number;
  tax_category: TaxCategory;
  tax_amount: number;
  net_amount: number;
  registration_number: string;
  proviso: string;
  issued_on: string;
};

export type CreateReceiptInput = {
  recipient: string;
  recipient_address?: string;
  amount: number;
  tax_category: TaxCategory;
  registration_number?: string;
  proviso?: string;
  issued_on?: string;
};

export type ReceiptFilter = {
  recipient?: string;
  issued_from?: string;
  issued_to?: string;
};
