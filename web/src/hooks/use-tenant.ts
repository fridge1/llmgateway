import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPost, apiPut, apiDelete } from "@/lib/api-client";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface TenantWithRole {
  id: string;
  name: string;
  role: string;
}

export interface Tenant {
  id: string;
  name: string;
  owner_id: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface TenantMember {
  id: string;
  tenant_id: string;
  user_id: string;
  role: string;
  joined_at: string;
  phone?: string;
  nickname?: string;
}

export interface TenantBalance {
  tenant_id?: string;
  user_id?: string;
  balance: number;
  frozen: number;
  total_recharged?: number;
  total_consumed?: number;
}

export interface TenantAPIKey {
  id: string;
  tenant_id: string;
  name: string;
  key_prefix: string;
  created_by: string;
  status: string;
  last_used_at: string | null;
  created_at: string;
}

export interface TenantTransaction {
  id: string;
  tenant_id: string;
  type: string;
  amount: number;
  balance_after: number;
  model: string;
  request_id: string;
  description: string;
  prompt_tokens?: number;
  completion_tokens?: number;
  cache_read_tokens?: number;
  cache_creation_tokens?: number;
  sub_user_id?: string;
  sub_user_username?: string;
  created_at: string;
}

export interface TenantInvitation {
  id: string;
  tenant_id: string;
  tenant_name?: string;
  phone: string;
  role: string;
  invited_by: string;
  status: string;
  expires_at: string;
  created_at: string;
}

export interface TenantSubUser {
  id: string;
  tenant_id: string;
  username: string;
  nickname: string;
  status: string;
  quota_limit: number | null;
  quota_used: number;
  quota_remaining: number | null;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface TenantTokenStats {
  total_prompt: number;
  total_completion: number;
  total_cache_read: number;
  total_cache_creation: number;
}

export interface SubUserCost {
  sub_user_id: string;
  sub_user_username: string;
  total_cost: number;
  request_count: number;
}

export interface TenantBillingStats {
  today_cost: number;
  month_cost: number;
  daily_average: number;
  daily_trend: { date: string; cost: number }[];
  model_breakdown: { model: string; cost: number }[];
  sub_user_ranking: SubUserCost[];
  token_stats: TenantTokenStats;
}

export interface SubUserModelCost {
  model: string;
  cost: number;
  request_count: number;
  prompt_tokens: number;
  completion_tokens: number;
  cache_read_tokens: number;
  cache_creation_tokens: number;
}

export interface SubUserModelStats {
  sub_user_id: string;
  sub_user_username: string;
  period?: {
    start_date: string;
    end_date: string;
  };
  total_cost: number;
  total_requests: number;
  model_breakdown: SubUserModelCost[];
}

// ---------------------------------------------------------------------------
// Query Keys
// ---------------------------------------------------------------------------

export const tenantKeys = {
  all: () => ["tenants"] as const,
  detail: (id: string) => ["tenants", id] as const,
  members: (id: string) => ["tenants", id, "members"] as const,
  balance: (id: string) => ["tenants", id, "balance"] as const,
  transactions: (id: string, page: number, size: number) =>
    ["tenants", id, "transactions", { page, size }] as const,
  keys: (id: string) => ["tenants", id, "keys"] as const,
  invitations: (id: string) => ["tenants", id, "invitations"] as const,
  pendingInvitations: () => ["invitations", "pending"] as const,
  subUsers: (id: string) => ["tenants", id, "sub-users"] as const,
  stats: (id: string, days: number) => ["tenants", id, "stats", days] as const,
  subUserTransactions: (tenantId: string, subUserId: string, page: number, size: number) =>
    ["tenants", tenantId, "sub-user-transactions", subUserId, { page, size }] as const,
  allSubUserTransactions: (id: string, page: number, size: number, subUserId?: string) =>
    ["tenants", id, "all-sub-user-transactions", { page, size, subUserId }] as const,
  subUserModelStats: (tenantId: string, subUserId: string, startDate?: string, endDate?: string) =>
    ["tenants", tenantId, "sub-user-model-stats", subUserId, { startDate, endDate }] as const,
};

// ---------------------------------------------------------------------------
// Queries
// ---------------------------------------------------------------------------

export function useTenants() {
  return useQuery({
    queryKey: tenantKeys.all(),
    queryFn: () => apiGet<TenantWithRole[]>("/api/tenants"),
  });
}

export function useTenantDetail(id: string) {
  return useQuery({
    queryKey: tenantKeys.detail(id),
    queryFn: () => apiGet<Tenant>(`/api/tenants/${id}`),
    enabled: !!id,
  });
}

export function useTenantMembers(id: string) {
  return useQuery({
    queryKey: tenantKeys.members(id),
    queryFn: () => apiGet<TenantMember[]>(`/api/tenants/${id}/members`),
    enabled: !!id,
  });
}

export function useTenantBalance(id: string) {
  return useQuery({
    queryKey: tenantKeys.balance(id),
    queryFn: () => apiGet<TenantBalance>(`/api/tenants/${id}/balance`),
    enabled: !!id,
  });
}

export function useTenantTransactions(id: string, page: number, size: number) {
  return useQuery({
    queryKey: tenantKeys.transactions(id, page, size),
    queryFn: () =>
      apiGet<{ transactions: TenantTransaction[]; total: number }>(
        `/api/tenants/${id}/transactions?limit=${size}&offset=${(page - 1) * size}`
      ),
    enabled: !!id,
  });
}

export function useTenantKeys(id: string) {
  return useQuery({
    queryKey: tenantKeys.keys(id),
    queryFn: () => apiGet<TenantAPIKey[]>(`/api/tenants/${id}/keys`),
    enabled: !!id,
  });
}

export function useTenantInvitations(id: string) {
  return useQuery({
    queryKey: tenantKeys.invitations(id),
    queryFn: () => apiGet<TenantInvitation[]>(`/api/tenants/${id}/invitations`),
    enabled: !!id,
  });
}

export function usePendingInvitations() {
  return useQuery({
    queryKey: tenantKeys.pendingInvitations(),
    queryFn: () => apiGet<TenantInvitation[]>("/api/invitations/pending"),
  });
}

export function useTenantSubUserTransactions(
  tenantId: string,
  subUserId: string,
  page: number,
  size: number
) {
  return useQuery({
    queryKey: tenantKeys.subUserTransactions(tenantId, subUserId, page, size),
    queryFn: () =>
      apiGet<{ transactions: TenantTransaction[]; total: number }>(
        `/api/tenants/${tenantId}/sub-users/${subUserId}/transactions?limit=${size}&offset=${(page - 1) * size}`
      ),
    enabled: !!tenantId && !!subUserId,
  });
}

export function useTenantAllSubUserTransactions(
  tenantId: string,
  page: number,
  size: number,
  subUserId?: string
) {
  return useQuery({
    queryKey: tenantKeys.allSubUserTransactions(tenantId, page, size, subUserId),
    queryFn: () => {
      let url = `/api/tenants/${tenantId}/all-transactions?limit=${size}&offset=${(page - 1) * size}`;
      if (subUserId) url += `&sub_user_id=${subUserId}`;
      return apiGet<{ transactions: TenantTransaction[]; total: number }>(url);
    },
    enabled: !!tenantId,
  });
}

export function useTenantSubUserModelStats(
  tenantId: string,
  subUserId: string,
  startDate?: string,
  endDate?: string
) {
  return useQuery({
    queryKey: tenantKeys.subUserModelStats(tenantId, subUserId, startDate, endDate),
    queryFn: () => {
      let url = `/api/tenants/${tenantId}/sub-users/${subUserId}/model-stats`;
      const params = new URLSearchParams();
      if (startDate) params.append("start_date", startDate);
      if (endDate) params.append("end_date", endDate);
      if (params.toString()) url += `?${params.toString()}`;
      return apiGet<SubUserModelStats>(url);
    },
    enabled: !!tenantId && !!subUserId,
  });
}

export function useTenantStats(tenantId: string, days: number) {
  return useQuery({
    queryKey: tenantKeys.stats(tenantId, days),
    queryFn: () =>
      apiGet<TenantBillingStats>(`/api/tenants/${tenantId}/stats?days=${days}`),
    enabled: !!tenantId,
  });
}

export async function exportTenantTransactions(
  tenantId: string,
  startDate?: string,
  endDate?: string,
  subUserID?: string
): Promise<void> {
  let url = `/api/tenants/${tenantId}/export-transactions?`;
  const params: string[] = [];
  if (startDate) params.push(`start_date=${startDate}`);
  if (endDate) params.push(`end_date=${endDate}`);
  if (subUserID) params.push(`sub_user_id=${subUserID}`);
  url += params.join("&");

  const res = await fetch(url, { credentials: "include" });
  if (!res.ok) throw new Error("Export failed");

  const blob = await res.blob();
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = res.headers.get("content-disposition")?.match(/filename="(.+)"/)?.[1] || "transactions.xlsx";
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(a.href);
}

// ---------------------------------------------------------------------------
// Mutations
// ---------------------------------------------------------------------------

export function useCreateTenant() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => apiPost<Tenant>("/api/tenants", { name }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.all() });
    },
  });
}

export function useUpdateTenant(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => apiPut(`/api/tenants/${id}`, { name }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.all() });
      qc.invalidateQueries({ queryKey: tenantKeys.detail(id) });
    },
  });
}

export function useDeleteTenant(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => apiDelete(`/api/tenants/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.all() });
    },
  });
}

export function useInviteMember(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { phone: string; role: string }) =>
      apiPost(`/api/tenants/${tenantId}/members/invite`, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.invitations(tenantId) });
      qc.invalidateQueries({ queryKey: tenantKeys.members(tenantId) });
    },
  });
}

export function useRemoveMember(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) =>
      apiDelete(`/api/tenants/${tenantId}/members/${userId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.members(tenantId) });
    },
  });
}

export function useUpdateMemberRole(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { userId: string; role: string }) =>
      apiPut(`/api/tenants/${tenantId}/members/${data.userId}/role`, { role: data.role }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.members(tenantId) });
    },
  });
}

export function useTransferOwnership(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (newOwnerID: string) =>
      apiPost(`/api/tenants/${tenantId}/transfer`, { new_owner_id: newOwnerID }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.all() });
      qc.invalidateQueries({ queryKey: tenantKeys.members(tenantId) });
    },
  });
}

export function useCreateTenantKey(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) =>
      apiPost<{ key: string }>(`/api/tenants/${tenantId}/keys`, { name }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.keys(tenantId) });
    },
  });
}

export function useDeleteTenantKey(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (keyId: string) =>
      apiDelete(`/api/tenants/${tenantId}/keys/${keyId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.keys(tenantId) });
    },
  });
}

export function useAcceptInvitation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (invitationId: string) =>
      apiPost(`/api/invitations/${invitationId}/accept`, {}),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.all() });
      qc.invalidateQueries({ queryKey: tenantKeys.pendingInvitations() });
    },
  });
}

// ---------------------------------------------------------------------------
// Sub-User Queries & Mutations
// ---------------------------------------------------------------------------

export function useTenantSubUsers(tenantId: string) {
  return useQuery({
    queryKey: tenantKeys.subUsers(tenantId),
    queryFn: () => apiGet<TenantSubUser[]>(`/api/tenants/${tenantId}/sub-users`),
    enabled: !!tenantId,
  });
}

export function useCreateSubUser(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { username: string; password: string; nickname?: string; quota_limit?: number | null }) =>
      apiPost(`/api/tenants/${tenantId}/sub-users`, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.subUsers(tenantId) });
    },
  });
}

export function useDeleteSubUser(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (subUserId: string) =>
      apiDelete(`/api/tenants/${tenantId}/sub-users/${subUserId}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.subUsers(tenantId) });
    },
  });
}

export function useResetSubUserPassword(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { subUserId: string; password: string }) =>
      apiPut(`/api/tenants/${tenantId}/sub-users/${data.subUserId}/password`, { new_password: data.password }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.subUsers(tenantId) });
    },
  });
}

export function useUpdateSubUserQuota(tenantId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { subUserId: string; quota_limit: number | null }) =>
      apiPut(`/api/tenants/${tenantId}/sub-users/${data.subUserId}/quota`, { quota_limit: data.quota_limit }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: tenantKeys.subUsers(tenantId) });
    },
  });
}
