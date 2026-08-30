// 組織(テナント) feature の型。バックエンドの tenantResponse / userResponse に対応する。

export type Tenant = {
  id: string;
  corporate_name: string;
  address: string;
  phone: string;
  email: string;
  registration_number: string;
};

export type UpdateTenantInput = {
  corporate_name: string;
  address: string;
  phone: string;
  email: string;
  registration_number: string;
};

export type Member = {
  user_id: string;
  email: string;
  display_name: string;
  is_active: boolean;
  tenant_id: string;
  role: string;
};

export type InviteMemberInput = {
  email: string;
  password: string;
};
