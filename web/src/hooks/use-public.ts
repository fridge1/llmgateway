import { useQuery } from "@tanstack/react-query";
import { apiGet } from "@/lib/api-client";
import type { PricingTier } from "@/types/api";

export interface PublicPlan {
  id: number;
  name: string;
  display_name: string;
  description: string;
  monthly_price_cny: number;
  quota_amount_cny: number;
  duration_days: number;
  sort_order: number;
  recommended: boolean;
}

export interface PublicStats {
  total_users: number;
  total_enterprises: number;
  total_requests: number;
  total_tokens: number;
  generated_at: string;
}

export interface PublicModel {
  id: string;
  display_name: string;
}

export interface ProviderGroup {
  provider: string;
  models: PublicModel[];
}

export const usePublicPlans = () =>
  useQuery({
    queryKey: ["public", "plans"],
    queryFn: () => apiGet<{ plans: PublicPlan[] }>("/api/public/plans"),
    staleTime: 10 * 60_000,
    retry: 0,
    refetchOnMount: false,
  });

export const usePublicStats = () =>
  useQuery({
    queryKey: ["public", "stats"],
    queryFn: () => apiGet<PublicStats>("/api/public/stats"),
    staleTime: 5 * 60_000,
    retry: 0,
  });

export const usePublicModels = () =>
  useQuery({
    queryKey: ["public", "models"],
    queryFn: () => apiGet<{ providers: ProviderGroup[] }>("/api/public/models"),
    staleTime: 30 * 60_000,
    retry: 0,
  });

export interface PublicPricingItem {
  model_name: string;
  display_name: string;
  input_price: number;
  output_price: number;
  cached_input_price: number;
  cache_creation_price: number;
  cache_creation_1h_price: number;
  billing_type: string;
  pricing_tiers?: PricingTier[];
}

export interface PricingProviderGroup {
  provider: string;
  items: PublicPricingItem[];
}

export const usePublicPricing = () =>
  useQuery({
    queryKey: ["public", "pricing"],
    queryFn: () =>
      apiGet<{ providers: PricingProviderGroup[] }>("/api/public/pricing"),
    staleTime: 30 * 60_000,
    retry: 0,
  });
