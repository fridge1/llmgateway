import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export const isWindows = navigator.userAgent.includes("Windows");

export function formatPrice(price: number): string {
  return price.toFixed(2);
}

// formatDiscount turns a multiplier rate (0,1] into a Chinese "折" label.
// 0.8 -> "8折", 0.85 -> "8.5折", 0.875 -> "8.75折".
export function formatDiscount(rate: number): string {
  const zhe = rate * 10;
  const rounded = Math.round(zhe * 100) / 100;
  return `${rounded}折`;
}

const NON_CHAT_CATEGORIES = new Set(["embedding", "text-to-image", "image-edit"]);

export function isChatModel(model: { category: string }): boolean {
  if (!model.category) return true;
  const cats = model.category.split(",").map((c) => c.trim());
  return cats.some((c) => !NON_CHAT_CATEGORIES.has(c));
}

