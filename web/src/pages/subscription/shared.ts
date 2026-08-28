import type { SubscriptionPlan } from "@/types/api";

export const tierColors: Record<
  string,
  { ring: string; badge: string; gradient: string; topBar: string }
> = {
  trial:   { ring: "ring-teal-400/40",    badge: "bg-teal-50 dark:bg-teal-500/10 text-teal-700 dark:text-teal-400 border-teal-200 dark:border-teal-500/20",       gradient: "from-teal-500 to-teal-600", topBar: "from-teal-400 to-teal-600" },
  pro:     { ring: "ring-indigo-400/40",   badge: "bg-indigo-50 dark:bg-indigo-500/10 text-indigo-700 dark:text-indigo-400 border-indigo-200 dark:border-indigo-500/20", gradient: "from-indigo-500 to-indigo-600", topBar: "from-indigo-400 to-indigo-600" },
  plus:    { ring: "ring-violet-400/40",   badge: "bg-violet-50 dark:bg-violet-500/10 text-violet-700 dark:text-violet-400 border-violet-200 dark:border-violet-500/20", gradient: "from-violet-500 to-violet-600", topBar: "from-violet-400 to-violet-600" },
  premium: { ring: "ring-emerald-400/40",  badge: "bg-emerald-50 dark:bg-emerald-500/10 text-emerald-700 dark:text-emerald-400 border-emerald-200 dark:border-emerald-500/20", gradient: "from-emerald-500 to-emerald-600", topBar: "from-emerald-400 to-emerald-600" },
  max:     { ring: "ring-amber-400/40",    badge: "bg-amber-50 dark:bg-amber-500/10 text-amber-700 dark:text-amber-400 border-amber-200 dark:border-amber-500/20",   gradient: "from-amber-500 to-amber-600", topBar: "from-amber-400 to-amber-600" },
  "openai-trial":   { ring: "ring-green-400/40",   badge: "bg-green-50 dark:bg-green-500/10 text-green-700 dark:text-green-400 border-green-200 dark:border-green-500/20",       gradient: "from-green-500 to-green-600",   topBar: "from-green-400 to-green-600" },
  "openai-pro":     { ring: "ring-lime-400/40",    badge: "bg-lime-50 dark:bg-lime-500/10 text-lime-700 dark:text-lime-400 border-lime-200 dark:border-lime-500/20",             gradient: "from-lime-500 to-lime-600",    topBar: "from-lime-400 to-lime-600" },
  "openai-plus":    { ring: "ring-cyan-400/40",    badge: "bg-cyan-50 dark:bg-cyan-500/10 text-cyan-700 dark:text-cyan-400 border-cyan-200 dark:border-cyan-500/20",             gradient: "from-cyan-500 to-cyan-600",    topBar: "from-cyan-400 to-cyan-600" },
  "openai-premium": { ring: "ring-sky-400/40",     badge: "bg-sky-50 dark:bg-sky-500/10 text-sky-700 dark:text-sky-400 border-sky-200 dark:border-sky-500/20",                   gradient: "from-sky-500 to-sky-600",     topBar: "from-sky-400 to-sky-600" },
  "openai-max":     { ring: "ring-rose-400/40",    badge: "bg-rose-50 dark:bg-rose-500/10 text-rose-700 dark:text-rose-400 border-rose-200 dark:border-rose-500/20",             gradient: "from-rose-500 to-rose-600",   topBar: "from-rose-400 to-rose-600" },
  "image-basic":    { ring: "ring-orange-400/40",  badge: "bg-orange-50 dark:bg-orange-500/10 text-orange-700 dark:text-orange-400 border-orange-200 dark:border-orange-500/20", gradient: "from-orange-500 to-orange-600", topBar: "from-orange-400 to-orange-600" },
  "image-pro":      { ring: "ring-pink-400/40",    badge: "bg-pink-50 dark:bg-pink-500/10 text-pink-700 dark:text-pink-400 border-pink-200 dark:border-pink-500/20",             gradient: "from-pink-500 to-pink-600",   topBar: "from-pink-400 to-pink-600" },
  "image-max":      { ring: "ring-fuchsia-400/40", badge: "bg-fuchsia-50 dark:bg-fuchsia-500/10 text-fuchsia-700 dark:text-fuchsia-400 border-fuchsia-200 dark:border-fuchsia-500/20", gradient: "from-fuchsia-500 to-fuchsia-600", topBar: "from-fuchsia-400 to-fuchsia-600" },
};

export type Brand = "claude" | "openai" | "image";

export const brandLabels: Record<Brand, string> = {
  claude: "Claude",
  openai: "OpenAI",
  image: "图片生成",
};

export const brandColors: Record<Brand, string> = {
  claude: "from-indigo-500 to-violet-500",
  openai: "from-green-500 to-emerald-500",
  image: "from-orange-500 to-fuchsia-500",
};

export const brandOrder: Brand[] = ["claude", "openai", "image"];

export const isImagePlan = (name: string) => name.startsWith("image-");

export const isTrialPlan = (name: string) =>
  name === "trial" || name === "openai-trial";

export const getBrand = (planName: string): Brand => {
  if (planName.startsWith("openai-")) return "openai";
  if (planName.startsWith("image-")) return "image";
  return "claude";
};

export const getBrandFromPlanId = (
  planId: number,
  plans: SubscriptionPlan[] | undefined,
): Brand => {
  const plan = plans?.find((p) => p.id === planId);
  return plan ? getBrand(plan.name) : "claude";
};

export const formatDate = (iso: string) =>
  new Date(iso).toLocaleDateString("zh-CN");

export const discountMap: Record<string, string> = {
  trial: "6.8折",
  pro: "5.8折",
  plus: "5.5折",
  premium: "5.2折",
  max: "5折",
  "openai-trial": "6.9折",
  "openai-pro": "7.7折",
  "openai-plus": "7.5折",
  "openai-premium": "7.5折",
  "openai-max": "6.7折",
};
