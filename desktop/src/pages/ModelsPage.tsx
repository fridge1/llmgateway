import { useState, useMemo } from "react";
import { useGatewayModels, usePricing } from "@/hooks/use-api";
import type { ModelPricing, PricingTier } from "@/lib/types-api";
import { Loader2 } from "../components/icons";
import { formatPrice, formatDiscount } from "@/lib/utils";

const ITEMS_PER_PAGE = 12;

const providerColors: Record<string, string> = {
  openai: "bg-emerald-500",
  anthropic: "bg-orange-400",
  google: "bg-blue-500",
  volcengine: "bg-red-400",
  xai: "bg-gray-400",
  deepseek: "bg-blue-400",
  siliconflow: "bg-purple-400",
  dashscope: "bg-yellow-400",
};

const categoryLabels: Record<string, string> = {
  reasoning: "推理",
  multimodal: "多模态",
  embedding: "向量嵌入",
  text: "文本",
  "text-to-image": "文生图",
  "image-edit": "图片编辑",
};

const categoryColors: Record<string, { bg: string; text: string }> = {
  reasoning: { bg: "bg-blue-900/40", text: "text-blue-300" },
  multimodal: { bg: "bg-purple-900/40", text: "text-purple-300" },
  embedding: { bg: "bg-green-900/40", text: "text-green-300" },
  text: { bg: "bg-gray-800/60", text: "text-gray-300" },
  "text-to-image": { bg: "bg-pink-900/40", text: "text-pink-300" },
  "image-edit": { bg: "bg-amber-900/40", text: "text-amber-300" },
};

function getProviderDot(provider: string) {
  return providerColors[provider.toLowerCase()] ?? "bg-gray-500";
}

function capitalize(s: string) {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function formatTokens(n: number) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(n % 1_000_000 === 0 ? 0 : 1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(n % 1_000 === 0 ? 0 : 1)}K`;
  return String(n);
}

function PriceValue({ value, original, unit, bold }: { value: number; original?: number; unit?: string; bold?: boolean }) {
  const showOriginal = original != null && formatPrice(original) !== formatPrice(value);
  return (
    <>
      {showOriginal && (
        <span className="text-obsidian-600 line-through mr-1">¥{formatPrice(original!)}</span>
      )}
      <span className={bold ? "text-obsidian-100 font-bold" : "text-obsidian-200 font-medium"}>¥{formatPrice(value)}{unit}</span>
    </>
  );
}

export default function ModelsPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const { data: gatewayModels, isLoading: modelsLoading } = useGatewayModels();
  const { data: pricingRes, isLoading: pricingLoading } = usePricing();

  const isLoading = modelsLoading || pricingLoading;

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

  const models = useMemo(() => {
    if (!gatewayModels) return [];
    return gatewayModels.map((m) => {
      const provider = m.upstreams?.[0]?.provider || "unknown";
      const price = pricingMap.get(m.name);
      const categories = m.category ? m.category.split(",").map((c) => c.trim()).filter(Boolean) : [];
      return { id: m.id, name: m.name, displayName: m.display_name || m.name, provider, categories, price };
    });
  }, [gatewayModels, pricingMap]);

  const providers = useMemo(() => {
    const set = new Set(models.map((m) => m.provider));
    return Array.from(set).sort();
  }, [models]);

  const allCategories = useMemo(() => {
    const set = new Set<string>();
    for (const m of models) for (const c of m.categories) set.add(c);
    return Array.from(set).sort();
  }, [models]);

  const filtered = useMemo(() => {
    let result = models.filter((m) => {
      const q = searchQuery.toLowerCase();
      const matchesSearch = !q || m.name.toLowerCase().includes(q) || m.displayName.toLowerCase().includes(q) || m.provider.toLowerCase().includes(q);
      const matchesProvider = !selectedProvider || m.provider === selectedProvider;
      const matchesCategory = !selectedCategory || m.categories.includes(selectedCategory);
      return matchesSearch && matchesProvider && matchesCategory;
    });
    result.sort((a, b) => {
      const aIsClaude = a.provider === "anthropic";
      const bIsClaude = b.provider === "anthropic";
      if (aIsClaude && !bIsClaude) return -1;
      if (!aIsClaude && bIsClaude) return 1;
      return a.name.localeCompare(b.name);
    });
    return result;
  }, [models, searchQuery, selectedProvider, selectedCategory]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / ITEMS_PER_PAGE));
  const currentPage = Math.min(page, totalPages);
  const paged = filtered.slice((currentPage - 1) * ITEMS_PER_PAGE, currentPage * ITEMS_PER_PAGE);
  const resetPage = () => setPage(1);

  if (isLoading) {
    return (
      <div className="p-6 flex items-center justify-center" style={{ minHeight: 400 }}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 size={24} className="animate-spin text-amber-400" />
          <span className="text-sm text-obsidian-400">加载模型列表...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-start justify-between mb-5">
        <div>
          <h1 className="text-lg font-semibold text-obsidian-50">可用模型</h1>
          <p className="text-xs text-obsidian-400 mt-0.5">
            {models.length} 个模型，来自 {providers.length} 个供应商
          </p>
        </div>
        <div className="relative w-56">
          <input
            className="w-full px-3 py-2 pl-8 bg-obsidian-800 border border-obsidian-700 rounded-lg text-sm text-obsidian-100 placeholder:text-obsidian-500 focus:outline-none focus:border-amber-500/50"
            placeholder="搜索模型..."
            value={searchQuery}
            onChange={(e) => { setSearchQuery(e.target.value); resetPage(); }}
          />
          <svg className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-obsidian-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-obsidian-900 border border-obsidian-700 rounded-xl p-4 mb-5 space-y-3">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs text-obsidian-400 font-medium mr-1">厂商</span>
          <FilterButton active={!selectedProvider} onClick={() => { setSelectedProvider(null); resetPage(); }}>全部</FilterButton>
          {providers.map((p) => (
            <FilterButton key={p} active={selectedProvider === p} onClick={() => { setSelectedProvider(selectedProvider === p ? null : p); resetPage(); }}>
              <span className={`w-2 h-2 rounded-full ${selectedProvider === p ? "bg-white" : getProviderDot(p)}`} />
              {capitalize(p)}
            </FilterButton>
          ))}
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs text-obsidian-400 font-medium mr-1">类型</span>
          <FilterButton active={!selectedCategory} onClick={() => { setSelectedCategory(null); resetPage(); }}>全部</FilterButton>
          {allCategories.map((c) => (
            <FilterButton key={c} active={selectedCategory === c} onClick={() => { setSelectedCategory(selectedCategory === c ? null : c); resetPage(); }}>
              {categoryLabels[c] || c}
            </FilterButton>
          ))}
        </div>
      </div>

      <div className="text-xs text-obsidian-400 mb-3">
        共 {filtered.length} 个模型，第 {currentPage} / {totalPages} 页
      </div>

      {paged.length === 0 ? (
        <div className="py-16 text-center">
          <div className="text-sm text-obsidian-300 mb-1">{models.length === 0 ? "平台正在配置中" : "暂无匹配模型"}</div>
          <div className="text-xs text-obsidian-500">{models.length === 0 ? "管理员尚未添加可用模型" : "请尝试其他搜索关键词或筛选条件"}</div>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-3 gap-3">
            {paged.map((model) => {
              const isClaude = model.provider === "anthropic";
              const isOpusOrSonnet = model.name.includes("opus") || model.name.includes("sonnet");
              return (
                <div
                  key={model.id}
                  className={`bg-obsidian-900 border rounded-xl p-4 hover:border-obsidian-600 transition-all duration-200 ${
                    isClaude ? "border-amber-500/30 ring-1 ring-amber-500/10" : "border-obsidian-700"
                  }`}
                >
                  <div className="flex items-center gap-1.5 mb-2">
                    <span className={`w-2 h-2 rounded-full ${getProviderDot(model.provider)}`} />
                    <span className="text-xs text-obsidian-400">{capitalize(model.provider)}</span>
                  </div>
                  <div className="text-sm font-semibold text-obsidian-50 mb-2">{model.displayName}</div>
                  <div className="flex gap-1.5 mb-3 flex-wrap">
                    {model.categories.map((cat) => {
                      const colors = categoryColors[cat] || { bg: "bg-gray-800/60", text: "text-gray-300" };
                      return (
                        <span key={cat} className={`px-1.5 py-0.5 rounded text-[10px] font-medium ${colors.bg} ${colors.text}`}>
                          {categoryLabels[cat] || cat}
                        </span>
                      );
                    })}
                    {isClaude && (
                      <>
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-900/40 text-blue-300">Prompt Cache</span>
                        {isOpusOrSonnet && (
                          <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-purple-900/40 text-purple-300">Extended Thinking</span>
                        )}
                      </>
                    )}
                  </div>
                  <div className="pt-2.5 border-t border-obsidian-800">
                    {model.price ? (
                      model.price.billing_type === "image" ? (
                        <div className="flex flex-wrap justify-between text-xs gap-y-1">
                          <div><span className="text-obsidian-400">1K/2K </span><PriceValue value={model.price.input_price} original={model.price.original_pricing?.input_price} unit="/张" /></div>
                          <div><span className="text-obsidian-400">4K </span><PriceValue value={model.price.output_price} original={model.price.original_pricing?.output_price} unit="/张" bold /></div>
                        </div>
                      ) : model.price.pricing_tiers && model.price.pricing_tiers.length > 0 ? (
                        <div className="text-xs space-y-1.5">
                          <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-900/40 text-blue-300">分梯次定价</span>
                          {model.price.pricing_tiers.map((tier, idx) => (
                            <div key={idx} className="flex items-start gap-2">
                              <span className="text-obsidian-400 whitespace-nowrap text-[10px]">{formatTokens(tier.min_tokens)}–{formatTokens(tier.max_tokens)}</span>
                              <span className="text-obsidian-200 font-medium text-[10px]">输入 ¥{tier.input_price}/M</span>
                              <span className="text-obsidian-100 font-bold text-[10px]">输出 ¥{tier.output_price}/M</span>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <div className="flex flex-wrap justify-between text-xs gap-y-1">
                          {model.price.discount_rate != null && (
                            <div className="w-full mb-1">
                              <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-900/40 text-amber-300">{formatDiscount(model.price.discount_rate)}</span>
                            </div>
                          )}
                          <div><span className="text-obsidian-400">输入 </span><PriceValue value={model.price.input_price} original={model.price.original_pricing?.input_price} unit="/M" /></div>
                          <div><span className="text-obsidian-400">输出 </span><PriceValue value={model.price.output_price} original={model.price.original_pricing?.output_price} unit="/M" bold /></div>
                          {model.price.cached_input_price > 0 && (
                            <div className="w-full"><span className="text-obsidian-400">缓存读取 </span><PriceValue value={model.price.cached_input_price} original={model.price.original_pricing?.cached_input_price} unit="/M" /></div>
                          )}
                          {model.price.cache_creation_price > 0 && (
                            <div className="w-full"><span className="text-obsidian-400">缓存写入(5min) </span><PriceValue value={model.price.cache_creation_price} original={model.price.original_pricing?.cache_creation_price} unit="/M" /></div>
                          )}
                          {model.price.cache_creation_1h_price > 0 && (
                            <div className="w-full"><span className="text-obsidian-400">缓存写入(1h) </span><PriceValue value={model.price.cache_creation_1h_price} original={model.price.original_pricing?.cache_creation_1h_price} unit="/M" /></div>
                          )}
                        </div>
                      )
                    ) : (
                      <div className="text-xs text-obsidian-500">暂无定价信息</div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 mt-5">
              <button
                className="px-3 py-1.5 rounded-lg text-xs font-medium bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
                disabled={currentPage <= 1}
                onClick={() => setPage(currentPage - 1)}
              >上一页</button>
              {Array.from({ length: totalPages }, (_, i) => i + 1).map((p) => (
                <button
                  key={p}
                  className={`w-7 h-7 rounded-lg text-xs font-medium transition-all duration-200 ${
                    p === currentPage
                      ? "bg-amber-500 text-obsidian-950"
                      : "bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700"
                  }`}
                  onClick={() => setPage(p)}
                >{p}</button>
              ))}
              <button
                className="px-3 py-1.5 rounded-lg text-xs font-medium bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
                disabled={currentPage >= totalPages}
                onClick={() => setPage(currentPage + 1)}
              >下一页</button>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function FilterButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      className={`px-2.5 py-1 rounded-full text-xs font-medium transition-all duration-200 flex items-center gap-1.5 ${
        active
          ? "bg-amber-500 text-obsidian-950"
          : "bg-obsidian-800 text-obsidian-300 hover:bg-obsidian-700"
      }`}
      onClick={onClick}
    >
      {children}
    </button>
  );
}
