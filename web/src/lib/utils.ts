import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatPrice(price: number): string {
  return price.toFixed(2);
}

// formatPricingFactor turns a pricing factor into a display label.
// < 1: discount (0.8 -> "8折", 0.85 -> "8.5折")
// = 1: original price ("原价")
// > 1: markup (1.2 -> "+20%", 1.5 -> "+50%")
export function formatPricingFactor(rate: number): string {
  if (rate < 1) {
    const zhe = rate * 10;
    const rounded = Math.round(zhe * 100) / 100;
    return `${rounded}折`;
  } else if (rate === 1) {
    return "原价";
  } else {
    const uplift = Math.round((rate - 1) * 1000) / 10;
    return `+${uplift}%`;
  }
}

// Backward compatibility alias
export const formatDiscount = formatPricingFactor;

const NON_CHAT_CATEGORIES = new Set(["embedding", "text-to-image", "image-edit"]);

export function isChatModel(model: { category: string }): boolean {
  if (!model.category) return true;
  const cats = model.category.split(",").map(c => c.trim());
  return cats.some(c => !NON_CHAT_CATEGORIES.has(c));
}
