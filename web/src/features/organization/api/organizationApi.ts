import { request } from '@/shared/api/httpClient';
import type {
  InviteMemberInput,
  Member,
  Tenant,
  UpdateTenantInput,
} from '@/features/organization/types';

// 認証トークンの付与・エラー処理は httpClient が担う。ここは薄い呼び出しだけ。

export function getTenant(): Promise<Tenant> {
  return request<Tenant>('/api/tenant');
}

export function updateTenant(input: UpdateTenantInput): Promise<Tenant> {
  return request<Tenant>('/api/tenant', {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}

export function listMembers(): Promise<Member[]> {
  return request<Member[]>('/api/members');
}

export function inviteMember(input: InviteMemberInput): Promise<Member> {
  return request<Member>('/api/members', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}
