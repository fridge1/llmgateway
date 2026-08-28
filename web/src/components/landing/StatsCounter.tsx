import { useEffect, useState } from "react";
import { usePublicStats, usePublicModels } from "@/hooks/use-public";
import { Skeleton } from "@/components/ui/skeleton";

function formatCompact(n: number) {
  if (n >= 1_000_000_000) return (n / 1_000_000_000).toFixed(1) + "B";
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + "M";
  if (n >= 1_000) return (n / 1_000).toFixed(1) + "K";
  return String(n);
}

function CountUp({ value, format }: { value: number; format: (n: number) => string }) {
  const [display, setDisplay] = useState(0);

  useEffect(() => {
    if (value <= 0) return;
    const start = performance.now();
    const duration = 900;
    let raf = 0;
    const tick = (t: number) => {
      const p = Math.min(1, (t - start) / duration);
      const eased = 1 - Math.pow(1 - p, 3);
      setDisplay(Math.round(value * eased));
      if (p < 1) raf = requestAnimationFrame(tick);
    };
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
  }, [value]);

  return <span>{format(display)}</span>;
}

interface StatItem {
  label: string;
  value: number;
  suffix?: string;
  format?: (n: number) => string;
}

const StatsCounter = () => {
  const { data, isLoading } = usePublicStats();
  const { data: modelsData } = usePublicModels();

  // 可调用模型数：跨上游分组求和，真实统计，不随用户量变化。
  const modelCount =
    modelsData?.providers?.reduce((sum, g) => sum + (g.models?.length ?? 0), 0) ?? 0;

  // 展示体量与能力，不展示用户规模：
  // 累计请求 / Token 为聚合体量（即便用户少也天然可观），
  // 可调用模型 / 兼容协议为平台能力（结构性指标，不露怯）。
  const items: StatItem[] = [
    { label: "累计请求", value: data?.total_requests ?? 0, format: formatCompact },
    { label: "累计 Token", value: data?.total_tokens ?? 0, format: formatCompact },
    { label: "可调用模型", value: modelCount, suffix: "+" },
    { label: "兼容主流协议", value: 3, suffix: " 种" },
  ];

  return (
    <section className="border-y border-border bg-card/40">
      <div className="max-w-6xl mx-auto px-6 py-12 grid grid-cols-2 md:grid-cols-4 gap-6">
        {items.map((item) => (
          <div key={item.label} className="text-center">
            {isLoading ? (
              <>
                <Skeleton className="h-9 w-20 mx-auto mb-2" />
                <Skeleton className="h-4 w-16 mx-auto" />
              </>
            ) : (
              <>
                <div className="text-3xl md:text-4xl font-bold tracking-tight brand-gradient-text">
                  <CountUp value={item.value} format={item.format ?? ((n) => String(n))} />
                  {item.suffix && <span>{item.suffix}</span>}
                </div>
                <div className="mt-1 text-sm text-muted-foreground">{item.label}</div>
              </>
            )}
          </div>
        ))}
      </div>
    </section>
  );
};

export default StatsCounter;
