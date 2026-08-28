import { useState } from "react";
import { usePublicPricing, type PublicPricingItem } from "@/hooks/use-public";
import { formatPrice } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";

import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from "@/components/ui/table";
const PROVIDER_LABEL: Record<string, string> = {
  openai: "OpenAI",
  anthropic: "Anthropic",
  google: "Google",
  gemini: "Google",
  volcengine: "火山引擎",
  xai: "xAI",
  deepseek: "DeepSeek",
  siliconflow: "SiliconFlow",
  dashscope: "阿里百炼",
  other: "其他",
};

function formatTokens(n: number) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(n % 1_000_000 === 0 ? 0 : 1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(n % 1_000 === 0 ? 0 : 1)}K`;
  return String(n);
}

function PricingCells({ item }: { item: PublicPricingItem }) {
  if (item.billing_type === "image") {
    return (
      <>
        <TableCell className="px-4 py-3 text-sm text-foreground">
          ¥{formatPrice(item.input_price)}
          <span className="text-muted-foreground"> /张 (1K/2K)</span>
        </TableCell>
        <TableCell className="px-4 py-3 text-sm text-foreground font-medium" colSpan={3}>
          ¥{formatPrice(item.output_price)}
          <span className="text-muted-foreground"> /张 (4K)</span>
        </TableCell>
      </>
    );
  }

  if (item.pricing_tiers && item.pricing_tiers.length > 0) {
    const tiers = item.pricing_tiers;
    const firstCached = tiers[0].cached_input_price;
    const lastCached = tiers[tiers.length - 1].cached_input_price;
    return (
      <TableCell className="px-4 py-3 text-sm" colSpan={4}>
        <div className="flex items-center gap-1.5 mb-1.5">
          <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300">
            分梯次定价
          </span>
        </div>
        <div className="space-y-1">
          {tiers.map((t, idx) => (
            <div key={idx} className="flex flex-wrap gap-x-4 gap-y-0.5 text-foreground">
              <span className="text-muted-foreground min-w-[80px]">
                {formatTokens(t.min_tokens)}–{formatTokens(t.max_tokens)}
              </span>
              <span>输入 ¥{formatPrice(t.input_price)}/M</span>
              <span className="font-medium">输出 ¥{formatPrice(t.output_price)}/M</span>
            </div>
          ))}
          {firstCached > 0 && (
            <div className="text-muted-foreground pt-0.5">
              缓存读取 ¥{formatPrice(firstCached)}/M
              {tiers.length > 1 && formatPrice(firstCached) !== formatPrice(lastCached) && (
                <span> ~ ¥{formatPrice(lastCached)}/M</span>
              )}
            </div>
          )}
        </div>
      </TableCell>
    );
  }

  return (
    <>
      <TableCell className="px-4 py-3 text-sm text-foreground">
        ¥{formatPrice(item.input_price)}
        <span className="text-muted-foreground">/M</span>
      </TableCell>
      <TableCell className="px-4 py-3 text-sm text-foreground font-medium">
        ¥{formatPrice(item.output_price)}
        <span className="text-muted-foreground">/M</span>
      </TableCell>
      <TableCell className="px-4 py-3 text-sm text-muted-foreground">
        {item.cached_input_price > 0 ? (
          <span className="text-foreground">¥{formatPrice(item.cached_input_price)}/M</span>
        ) : (
          "—"
        )}
      </TableCell>
      <TableCell className="px-4 py-3 text-sm text-muted-foreground">
        {item.cache_creation_price > 0 || item.cache_creation_1h_price > 0 ? (
          <div className="space-y-0.5">
            {item.cache_creation_price > 0 && (
              <div className="text-foreground">
                ¥{formatPrice(item.cache_creation_price)}
                <span className="text-muted-foreground">/M (5min)</span>
              </div>
            )}
            {item.cache_creation_1h_price > 0 && (
              <div className="text-foreground">
                ¥{formatPrice(item.cache_creation_1h_price)}
                <span className="text-muted-foreground">/M (1h)</span>
              </div>
            )}
          </div>
        ) : (
          "—"
        )}
      </TableCell>
    </>
  );
}

const ModelPricingTable = () => {
  const { data, isLoading } = usePublicPricing();
  const providers = data?.providers ?? [];
  const [activeIdx, setActiveIdx] = useState(0);

  const active = providers[activeIdx];

  return (
    <section id="pricing" className="py-20 bg-muted/30">
      <div className="max-w-6xl mx-auto px-6">
        <div className="text-center mb-12">
          <h2 className="text-3xl md:text-4xl font-bold text-foreground tracking-tight mb-3">
            模型价格
          </h2>
          <p className="text-muted-foreground">
            按 token 计费，支持缓存命中折扣，所有价格以人民币结算
          </p>
        </div>

        {isLoading ? (
          <div className="space-y-4">
            <div className="flex justify-center gap-2">
              <Skeleton className="h-9 w-24" />
              <Skeleton className="h-9 w-24" />
              <Skeleton className="h-9 w-24" />
            </div>
            <Skeleton className="h-80 rounded-xl" />
          </div>
        ) : providers.length === 0 ? (
          <p className="text-center text-muted-foreground">暂无定价信息</p>
        ) : (
          <>
            <div className="flex flex-wrap justify-center gap-2 mb-8">
              {providers.map((p, idx) => (
                <button
                  key={p.provider}
                  onClick={() => setActiveIdx(idx)}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                    idx === activeIdx
                      ? "bg-primary text-primary-foreground shadow-sm"
                      : "bg-muted/60 text-muted-foreground hover:bg-muted"
                  }`}
                >
                  {PROVIDER_LABEL[p.provider] ?? p.provider}
                  <span className="ml-1.5 text-xs opacity-70">{p.items.length}</span>
                </button>
              ))}
            </div>

            <div className="bg-card border border-border rounded-xl shadow-card overflow-hidden">
              <Table className="w-full min-w-[720px]">
                  <TableHeader>
                    <TableRow className="border-b border-border bg-muted/40">
                      <TableHead className="text-left px-4 py-3 text-sm font-semibold text-foreground">模型</TableHead>
                      <TableHead className="text-left px-4 py-3 text-sm font-semibold text-foreground">输入</TableHead>
                      <TableHead className="text-left px-4 py-3 text-sm font-semibold text-foreground">输出</TableHead>
                      <TableHead className="text-left px-4 py-3 text-sm font-semibold text-foreground">缓存读取</TableHead>
                      <TableHead className="text-left px-4 py-3 text-sm font-semibold text-foreground">缓存写入</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {active?.items.map((item) => (
                      <TableRow
                        key={item.model_name}
                        className="border-b border-border last:border-0 hover:bg-muted/20 transition-colors"
                      >
                        <TableCell className="px-4 py-3">
                          <div className="font-medium text-sm text-foreground">{item.display_name}</div>
                          <div className="text-xs text-muted-foreground mt-0.5">{item.model_name}</div>
                        </TableCell>
                        <PricingCells item={item} />
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                          </div>

            <div className="mt-4 text-center text-xs text-muted-foreground">
              * 价格单位为人民币（CNY），按每百万 tokens 计费；图片模型按张计费
            </div>
          </>
        )}
      </div>
    </section>
  );
};

export default ModelPricingTable;
