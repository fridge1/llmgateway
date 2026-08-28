import { useState, useMemo } from "react";
import { Search, Loader2 } from "lucide-react";
import { useGatewayModels, usePricing } from "@/hooks/use-api";
import { formatPrice, formatPricingFactor, cn } from "@/lib/utils";
import { PageHeader } from "@/components/ui/page-header";
import type { PricingTier, ModelPricing } from "@/types/api";

const ITEMS_PER_PAGE = 12;

const providerColors: Record<string, string> = {
  openai: "bg-emerald-500",
  anthropic: "bg-orange-400 dark:bg-orange-300",
  google: "bg-blue-500",
  volcengine: "bg-red-500 dark:bg-red-400",
  xai: "bg-gray-500 dark:bg-gray-400",
  deepseek: "bg-blue-600 dark:bg-blue-400",
  siliconflow: "bg-purple-500 dark:bg-purple-400",
  dashscope: "bg-yellow-500 dark:bg-yellow-400",
};

const categoryLabels: Record<string, string> = {
  reasoning: "\u63A8\u7406",
  multimodal: "\u591A\u6A21\u6001",
  embedding: "\u5411\u91CF\u5D4C\u5165",
  text: "\u6587\u672C",
  "text-to-image": "\u6587\u751F\u56FE",
  "image-edit": "\u56FE\u7247\u7F16\u8F91",
};

const categoryColors: Record<string, { bg: string; text: string }> = {
  reasoning: { bg: "bg-blue-100 dark:bg-blue-900/40", text: "text-blue-700 dark:text-blue-300" },
  multimodal: { bg: "bg-purple-100 dark:bg-purple-900/40", text: "text-purple-700 dark:text-purple-300" },
  embedding: { bg: "bg-green-100 dark:bg-green-900/40", text: "text-green-700 dark:text-green-300" },
  text: { bg: "bg-gray-100 dark:bg-gray-800/60", text: "text-gray-700 dark:text-gray-300" },
  "text-to-image": { bg: "bg-pink-100 dark:bg-pink-900/40", text: "text-pink-700 dark:text-pink-300" },
  "image-edit": { bg: "bg-amber-100 dark:bg-amber-900/40", text: "text-amber-700 dark:text-amber-300" },
};

function getProviderDot(provider: string) {
  return providerColors[provider.toLowerCase()] ?? "bg-gray-400 dark:bg-gray-500";
}

function capitalize(s: string) {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function formatTokens(n: number) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(n % 1_000_000 === 0 ? 0 : 1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(n % 1_000 === 0 ? 0 : 1)}K`;
  return String(n);
}

// PriceValue renders a price, optionally preceded by a struck-through original
// price when a tenant discount applies (and the original differs after rounding).
function PriceValue({ value, original, unit, bold }: { value: number; original?: number; unit?: string; bold?: boolean }) {
  const showOriginal = original != null && formatPrice(original) !== formatPrice(value);
  return (
    <>
      {showOriginal && (
        <span className="text-muted-foreground line-through mr-1">{"¥"}{formatPrice(original)}</span>
      )}
      <span className={`text-foreground ${bold ? "font-bold" : "font-medium"}`}>{"¥"}{formatPrice(value)}{unit}</span>
    </>
  );
}

const ModelsPage = () => {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const { data: gatewayModels, isLoading: modelsLoading } = useGatewayModels();
  const { data: pricingRes, isLoading: pricingLoading } = usePricing();

  const isLoading = modelsLoading || pricingLoading;

  // Build a pricing lookup map
  const pricingMap = useMemo(() => {
    const map = new Map<string, { input_price: number; output_price: number; cached_input_price: number; cache_creation_price: number; cache_creation_1h_price: number; billing_type: string; pricing_tiers?: PricingTier[]; discount_rate?: number; original_pricing?: ModelPricing }>();
    const rows = pricingRes?.pricing;
    if (rows) {
      for (const p of rows) {
        map.set(p.model_name, {
          input_price: p.input_price,
          output_price: p.output_price,
          cached_input_price: p.cached_input_price,
          cache_creation_price: p.cache_creation_price,
          cache_creation_1h_price: p.cache_creation_1h_price,
          billing_type: p.billing_type || "token",
          pricing_tiers: p.pricing_tiers,
          discount_rate: p.discount_rate,
          original_pricing: p.original_pricing,
        });
      }
    }
    return map;
  }, [pricingRes]);

  // Build the model list
  const models = useMemo(() => {
    if (!gatewayModels) return [];
    return gatewayModels.map((m) => {
      const provider = m.upstreams?.[0]?.provider || "unknown";
      const price = pricingMap.get(m.name);
      const categories = m.category ? m.category.split(",").map((c) => c.trim()).filter(Boolean) : [];
      return {
        id: m.id,
        name: m.name,
        displayName: m.display_name || m.name,
        provider,
        categories,
        price,
      };
    });
  }, [gatewayModels, pricingMap]);

  // Unique providers for filter
  const providers = useMemo(() => {
    const set = new Set(models.map((m) => m.provider));
    return Array.from(set).sort();
  }, [models]);

  // Unique categories for filter
  const allCategories = useMemo(() => {
    const set = new Set<string>();
    for (const m of models) {
      for (const c of m.categories) set.add(c);
    }
    return Array.from(set).sort();
  }, [models]);

  // Filter
  const filtered = useMemo(() => {
    const result = models.filter((m) => {
      const q = searchQuery.toLowerCase();
      const matchesSearch =
        !q ||
        m.name.toLowerCase().includes(q) ||
        m.displayName.toLowerCase().includes(q) ||
        m.provider.toLowerCase().includes(q);
      const matchesProvider = !selectedProvider || m.provider === selectedProvider;
      const matchesCategory = !selectedCategory || m.categories.includes(selectedCategory);
      return matchesSearch && matchesProvider && matchesCategory;
    });

    // Sort: Claude models first, then others
    result.sort((a, b) => {
      const aIsClaude = a.provider === "anthropic";
      const bIsClaude = b.provider === "anthropic";
      if (aIsClaude && !bIsClaude) return -1;
      if (!aIsClaude && bIsClaude) return 1;
      return a.name.localeCompare(b.name);
    });

    return result;
  }, [models, searchQuery, selectedProvider, selectedCategory]);

  // Pagination
  const totalPages = Math.max(1, Math.ceil(filtered.length / ITEMS_PER_PAGE));
  const currentPage = Math.min(page, totalPages);
  const paged = filtered.slice((currentPage - 1) * ITEMS_PER_PAGE, currentPage * ITEMS_PER_PAGE);

  // Reset page when filters change
  const resetPage = () => setPage(1);

  return (
    <div className="page-container fade-in">
      <PageHeader
        eyebrow="模型"
        title={"\u53EF\u7528\u6A21\u578B"}
        description={`共 ${models.length} 个模型，来自 ${providers.length} 个供应商。按厂商与品类筛选，查看定价与上下文窗口。`}
        actions={
          <div className="relative w-64">
            <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input
              className="input-field"
              style={{ paddingLeft: "2.25rem" }}
              placeholder={"\u641C\u7D22\u6A21\u578B..."}
              value={searchQuery}
              onChange={(e) => { setSearchQuery(e.target.value); resetPage(); }}
            />
          </div>
        }
      />

      {/* Filters */}
      <div className="bg-card border border-border rounded-xl p-5 mb-6 shadow-card space-y-4">
        {/* Provider filter */}
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm text-muted-foreground font-medium mr-2">{"\u5382\u5546"}</span>
          <button
            className={`px-3 py-1 rounded-full text-sm font-medium transition-all duration-200 ${
              !selectedProvider
                ? "bg-gradient-to-r from-[var(--brand-gradient-start)] to-[var(--brand-gradient-end)] text-white shadow-button"
                : "bg-muted text-muted-foreground hover:bg-muted/80"
            }`}
            onClick={() => {
              setSelectedProvider(null);
              resetPage();
            }}
          >
            {"\u5168\u90E8"}
          </button>
          {providers.map((p) => (
            <button
              key={p}
              className={`px-3 py-1 rounded-full text-sm font-medium transition-all duration-200 flex items-center gap-1.5 ${
                selectedProvider === p
                  ? "bg-gradient-to-r from-[var(--brand-gradient-start)] to-[var(--brand-gradient-end)] text-white shadow-button"
                  : "bg-muted text-muted-foreground hover:bg-muted/80"
              }`}
              onClick={() => {
                setSelectedProvider(selectedProvider === p ? null : p);
                resetPage();
              }}
            >
              <span className={`w-2 h-2 rounded-full ${selectedProvider === p ? "bg-white" : getProviderDot(p)}`} />
              {capitalize(p)}
            </button>
          ))}
        </div>

        {/* Category filter */}
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm text-muted-foreground font-medium mr-2">{"\u6A21\u578B\u7C7B\u578B"}</span>
          <button
            className={`px-3 py-1 rounded-full text-sm font-medium transition-all duration-200 ${
              !selectedCategory
                ? "bg-gradient-to-r from-[var(--brand-gradient-start)] to-[var(--brand-gradient-end)] text-white shadow-button"
                : "bg-muted text-muted-foreground hover:bg-muted/80"
            }`}
            onClick={() => {
              setSelectedCategory(null);
              resetPage();
            }}
          >
            {"\u5168\u90E8"}
          </button>
          {allCategories.map((c) => (
            <button
              key={c}
              className={`px-3 py-1 rounded-full text-sm font-medium transition-all duration-200 ${
                selectedCategory === c
                  ? "bg-gradient-to-r from-[var(--brand-gradient-start)] to-[var(--brand-gradient-end)] text-white shadow-button"
                  : "bg-muted text-muted-foreground hover:bg-muted/80"
              }`}
              onClick={() => {
                setSelectedCategory(selectedCategory === c ? null : c);
                resetPage();
              }}
            >
              {categoryLabels[c] || c}
            </button>
          ))}
        </div>
      </div>

      {/* Pagination info */}
      <div className="text-sm text-muted-foreground mb-4">
        {"\u5171"} {filtered.length} {"\u4E2A\u6A21\u578B\uFF0C\u7B2C"} {currentPage} / {totalPages} {"\u9875"}
      </div>

      {/* Loading state */}
      {isLoading ? (
        <div className="empty-state bg-card border border-border rounded-xl py-24 shadow-card">
          <Loader2 size={28} className="text-primary animate-spin mb-4" />
          <div className="text-sm font-semibold text-foreground">{"\u52A0\u8F7D\u6A21\u578B\u5217\u8868..."}</div>
        </div>
      ) : paged.length === 0 ? (
        <div className="empty-state bg-card border border-border rounded-xl py-24 shadow-card">
          {models.length === 0 ? (
            <>
              <div className="text-sm font-semibold text-foreground mb-1">{"\u5E73\u53F0\u6B63\u5728\u914D\u7F6E\u4E2D"}</div>
              <div className="text-xs text-muted-foreground">{"\u7BA1\u7406\u5458\u5C1A\u672A\u6DFB\u52A0\u53EF\u7528\u6A21\u578B\uFF0C\u8BF7\u7A0D\u540E\u518D\u8BD5"}</div>
            </>
          ) : (
            <>
              <div className="text-sm font-semibold text-foreground mb-1">{"\u6682\u65E0\u5339\u914D\u6A21\u578B"}</div>
              <div className="text-xs text-muted-foreground">{"\u8BF7\u5C1D\u8BD5\u5176\u4ED6\u641C\u7D22\u5173\u952E\u8BCD\u6216\u7B5B\u9009\u6761\u4EF6"}</div>
            </>
          )}
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {paged.map((model, i) => {
              const isClaude = model.provider === "anthropic";
              const isOpusOrSonnet = model.name.includes("opus") || model.name.includes("sonnet");

              return (
              <div
                key={model.id}
                className={`stagger-item bg-card border rounded-xl p-5 shadow-card hover:shadow-elevated hover:-translate-y-0.5 transition-all duration-300 cursor-pointer ${
                  isClaude ? "border-primary/40 ring-1 ring-primary/20" : "border-border"
                }`}
                style={{ animationDelay: `${i * 80}ms` }}
              >
                {/* Provider */}
                <div className="flex items-center gap-1.5 mb-2">
                  <span className={`w-2.5 h-2.5 rounded-full ${getProviderDot(model.provider)}`} />
                  <span className="text-sm text-muted-foreground">{capitalize(model.provider)}</span>
                </div>

                {/* Model name */}
                <div className="text-base font-bold text-foreground mb-2">{model.displayName}</div>

                {/* Category badges + Claude feature tags */}
                <div className="flex gap-1.5 mb-4 flex-wrap">
                  {model.categories.map((cat) => {
                    const colors = categoryColors[cat] || { bg: "bg-gray-100 dark:bg-gray-800/60", text: "text-gray-700 dark:text-gray-300" };
                    return (
                      <span
                        key={cat}
                        className={`px-2 py-0.5 rounded text-xs font-medium ${colors.bg} ${colors.text}`}
                      >
                        {categoryLabels[cat] || cat}
                      </span>
                    );
                  })}
                  {isClaude && (
                    <>
                      <span className="px-2 py-0.5 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300 border border-blue-200 dark:border-blue-700/50">
                        Prompt Cache
                      </span>
                      {isOpusOrSonnet && (
                        <span className="px-2 py-0.5 rounded text-xs font-medium bg-purple-100 dark:bg-purple-900/40 text-purple-700 dark:text-purple-300 border border-purple-200 dark:border-purple-700/50">
                          Extended Thinking
                        </span>
                      )}
                    </>
                  )}
                </div>

                {/* Pricing */}
                <div className="pt-3 border-t border-border">
                  {model.price ? (() => {
                    const orig = model.price.discount_rate != null ? model.price.original_pricing : undefined;
                    const discountBadge = model.price.discount_rate != null && model.price.discount_rate !== 1 ? (
                      <span className={cn(
                        "px-1.5 py-0.5 rounded text-[10px] font-medium",
                        model.price.discount_rate < 1
                          ? "bg-rose-100 dark:bg-rose-900/40 text-rose-700 dark:text-rose-300"
                          : "bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300"
                      )}>
                        {formatPricingFactor(model.price.discount_rate)}
                      </span>
                    ) : null;
                    return (
                    model.price.billing_type === "image" ? (
                      <div className="flex flex-wrap justify-between items-center text-xs gap-y-1">
                        {discountBadge && <div className="w-full mb-1">{discountBadge}</div>}
                        <div>
                          <span className="text-muted-foreground">1K/2K </span>
                          <PriceValue value={model.price.input_price} original={orig?.input_price} unit={"\u002F\u5F20"} />
                        </div>
                        <div>
                          <span className="text-muted-foreground">4K </span>
                          <PriceValue value={model.price.output_price} original={orig?.output_price} unit={"\u002F\u5F20"} bold />
                        </div>
                      </div>
                    ) : model.price.pricing_tiers && model.price.pricing_tiers.length > 0 ? (
                    <div className="text-xs space-y-2">
                      <div className="flex items-center gap-1.5 mb-1">
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300">{"\u5206\u68AF\u6B21\u5B9A\u4EF7"}</span>
                        {discountBadge}
                      </div>
                      {model.price.pricing_tiers.map((tier, idx) => (
                        <div key={idx} className="flex items-start gap-2">
                          <span className="text-muted-foreground whitespace-nowrap min-w-[70px]">
                            {formatTokens(tier.min_tokens)}{"\u2013"}{formatTokens(tier.max_tokens)}
                          </span>
                          <span>
                            {"\u8F93\u5165"} <PriceValue value={tier.input_price} original={orig?.pricing_tiers?.[idx]?.input_price} unit="/M" />
                          </span>
                          <span>
                            {"\u8F93\u51FA"} <PriceValue value={tier.output_price} original={orig?.pricing_tiers?.[idx]?.output_price} unit="/M" bold />
                          </span>
                        </div>
                      ))}
                      {model.price.pricing_tiers[0].cached_input_price > 0 && (
                        <div className="pt-1 border-t border-border/50">
                          <span className="text-muted-foreground">{"\u7F13\u5B58\u8BFB\u53D6"} </span>
                          <PriceValue value={model.price.pricing_tiers[0].cached_input_price} original={orig?.pricing_tiers?.[0]?.cached_input_price} unit="/M" />
                          {model.price.pricing_tiers.length > 1 && formatPrice(model.price.pricing_tiers[0].cached_input_price) !== formatPrice(model.price.pricing_tiers[model.price.pricing_tiers.length - 1].cached_input_price) && (
                            <span className="text-muted-foreground"> ~ {"\u00A5"}{formatPrice(model.price.pricing_tiers[model.price.pricing_tiers.length - 1].cached_input_price)}/M</span>
                          )}
                        </div>
                      )}
                    </div>
                    ) : (
                    <div className="flex flex-wrap justify-between text-xs gap-y-1">
                      {discountBadge && <div className="w-full mb-1">{discountBadge}</div>}
                      <div>
                        <span className="text-muted-foreground">{"\u8F93\u5165"} </span>
                        <PriceValue value={model.price.input_price} original={orig?.input_price} unit="/M" />
                      </div>
                      <div>
                        <span className="text-muted-foreground">{"\u8F93\u51FA"} </span>
                        <PriceValue value={model.price.output_price} original={orig?.output_price} unit="/M" bold />
                      </div>
                      {model.price.cached_input_price > 0 && (
                        <div className="w-full">
                          <span className="text-muted-foreground">{"\u7F13\u5B58\u8BFB\u53D6"} </span>
                          <PriceValue value={model.price.cached_input_price} original={orig?.cached_input_price} unit="/M" />
                        </div>
                      )}
                      {model.price.cache_creation_price > 0 && (
                        <div className="w-full">
                          <span className="text-muted-foreground">{"\u7F13\u5B58\u5199\u5165"}(5min) </span>
                          <PriceValue value={model.price.cache_creation_price} original={orig?.cache_creation_price} unit="/M" />
                        </div>
                      )}
                      {model.price.cache_creation_1h_price > 0 && (
                        <div className="w-full">
                          <span className="text-muted-foreground">{"\u7F13\u5B58\u5199\u5165"}(1h) </span>
                          <PriceValue value={model.price.cache_creation_1h_price} original={orig?.cache_creation_1h_price} unit="/M" />
                        </div>
                      )}
                    </div>
                    )
                    );
                  })() : (
                    <div className="text-xs text-muted-foreground">{"\u6682\u65E0\u5B9A\u4EF7\u4FE1\u606F"}</div>
                  )}
                </div>
              </div>
              );
            })}
          </div>

          {/* Pagination controls */}
          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-6">
              <button
                className="px-3 py-1.5 rounded-lg text-sm font-medium bg-muted text-muted-foreground hover:bg-muted/80 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
                disabled={currentPage <= 1}
                onClick={() => setPage(currentPage - 1)}
              >
                {"\u4E0A\u4E00\u9875"}
              </button>
              {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
                <button
                  key={p}
                  className={`w-8 h-8 rounded-lg text-sm font-medium transition-all duration-200 ${
                    p === currentPage
                      ? "bg-gradient-to-r from-[var(--brand-gradient-start)] to-[var(--brand-gradient-end)] text-white shadow-button"
                      : "bg-muted text-muted-foreground hover:bg-muted/80"
                  }`}
                  onClick={() => setPage(p)}
                >
                  {p}
                </button>
              ))}
              <button
                className="px-3 py-1.5 rounded-lg text-sm font-medium bg-muted text-muted-foreground hover:bg-muted/80 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
                disabled={currentPage >= totalPages}
                onClick={() => setPage(currentPage + 1)}
              >
                {"\u4E0B\u4E00\u9875"}
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default ModelsPage;
