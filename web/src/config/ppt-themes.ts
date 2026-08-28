import type { PptConfig } from "@/types/ppt";

export type ThemeMotif =
  | "side-bar"
  | "card-grid"
  | "split-bleed"
  | "rounded-block"
  | "editorial-serif"
  | "minimal-none";

export type ThemeMode = "light" | "dark";

export type AccentStrategy =
  | "numeric-only"
  | "heading-color"
  | "block"
  | "underline-num";

export interface ThemePalette {
  bg: string;
  surface: string;
  text: string;
  textMuted: string;
  primary: string;
  accent: string;
  accentSoft: string;
  rule: string;
}

export interface ThemeChartPalette {
  categorical: string[];
  sequential: string[];
  diverging: [string, string, string];
}

export interface ThemeFontPair {
  heading: string;
  body: string;
}

export interface PptTheme {
  // Identity
  id: string;
  name: string;
  mode: ThemeMode;
  motif: ThemeMotif;
  accentStrategy: AccentStrategy;

  // New design system (Phase 2/3)
  palette: ThemePalette;
  chartPalette: ThemeChartPalette;
  fontPair: ThemeFontPair;

  // ----- Legacy fields kept populated for back-compat with current exporter/preview -----
  primary: string;
  primaryHex: string;
  secondary: string;
  secondaryHex: string;
  bg: string;
  bgHex: string;
  text: string;
  textHex: string;
  accent: string;
  accentHex: string;
  chartColors?: string[];
  fontFamily: string;
  headingSize: number;
  bodySize: number;
  showPageNumber: boolean;
  decorationLine: boolean;
}

export const pptThemes: PptTheme[] = [
  {
    id: "midnight-executive",
    name: "深空高管",
    mode: "dark",
    motif: "card-grid",
    accentStrategy: "numeric-only",
    palette: {
      bg: "#0B1220",
      surface: "#111A2C",
      text: "#E6EDF7",
      textMuted: "#7C8AA3",
      primary: "#0B1220",
      accent: "#3DDBD9",
      accentSoft: "#1A2E3A",
      rule: "#1F2D44",
    },
    chartPalette: {
      categorical: ["#3DDBD9", "#FFB86B", "#C792EA", "#82AAFF", "#F07178", "#FFCB6B"],
      sequential: ["#0B1220", "#142036", "#1F3354", "#2C4A78", "#3DDBD9"],
      diverging: ["#F07178", "#7C8AA3", "#3DDBD9"],
    },
    fontPair: { heading: "Georgia", body: "Calibri" },
    primary: "bg-slate-900",
    primaryHex: "0B1220",
    secondary: "bg-slate-800",
    secondaryHex: "111A2C",
    bg: "bg-slate-950",
    bgHex: "0B1220",
    text: "text-slate-100",
    textHex: "E6EDF7",
    accent: "text-cyan-300",
    accentHex: "3DDBD9",
    chartColors: ["#3DDBD9", "#FFB86B", "#C792EA", "#82AAFF", "#F07178", "#FFCB6B"],
    fontFamily: "Georgia",
    headingSize: 26,
    bodySize: 16,
    showPageNumber: true,
    decorationLine: false,
  },
  {
    id: "warm-terracotta",
    name: "暖陶土",
    mode: "light",
    motif: "split-bleed",
    accentStrategy: "heading-color",
    palette: {
      bg: "#FBF7F2",
      surface: "#FFFFFF",
      text: "#2A201B",
      textMuted: "#8C7A6E",
      primary: "#C8553D",
      accent: "#5C8A8A",
      accentSoft: "#E7D9CE",
      rule: "#E2D5C7",
    },
    chartPalette: {
      categorical: ["#C8553D", "#5C8A8A", "#E2A65A", "#2F5061", "#6BAA75", "#BFAE48"],
      sequential: ["#FBF7F2", "#F0DACE", "#E2A65A", "#C8553D", "#7A2E1E"],
      diverging: ["#C8553D", "#FBF7F2", "#5C8A8A"],
    },
    fontPair: { heading: "Playfair Display", body: "Inter" },
    primary: "bg-orange-700",
    primaryHex: "C8553D",
    secondary: "bg-orange-100",
    secondaryHex: "E7D9CE",
    bg: "bg-stone-50",
    bgHex: "FBF7F2",
    text: "text-stone-900",
    textHex: "2A201B",
    accent: "text-teal-700",
    accentHex: "5C8A8A",
    chartColors: ["#C8553D", "#5C8A8A", "#E2A65A", "#2F5061", "#6BAA75", "#BFAE48"],
    fontFamily: "Playfair Display, Georgia, serif",
    headingSize: 26,
    bodySize: 16,
    showPageNumber: true,
    decorationLine: false,
  },
  {
    id: "charcoal-minimal",
    name: "极简白",
    mode: "light",
    motif: "minimal-none",
    accentStrategy: "numeric-only",
    palette: {
      bg: "#FFFFFF",
      surface: "#FAFAFA",
      text: "#111111",
      textMuted: "#666666",
      primary: "#111111",
      accent: "#111111",
      accentSoft: "#F2F2F2",
      rule: "#E5E5E5",
    },
    chartPalette: {
      categorical: ["#111111", "#6E6E6E", "#B5B5B5", "#2563EB", "#16A34A", "#DC2626"],
      sequential: ["#FFFFFF", "#E5E5E5", "#B5B5B5", "#6E6E6E", "#111111"],
      diverging: ["#DC2626", "#F2F2F2", "#2563EB"],
    },
    fontPair: { heading: "Inter", body: "Inter" },
    primary: "bg-neutral-900",
    primaryHex: "111111",
    secondary: "bg-neutral-100",
    secondaryHex: "F2F2F2",
    bg: "bg-white",
    bgHex: "FFFFFF",
    text: "text-neutral-900",
    textHex: "111111",
    accent: "text-neutral-900",
    accentHex: "111111",
    chartColors: ["#111111", "#6E6E6E", "#B5B5B5", "#2563EB", "#16A34A", "#DC2626"],
    fontFamily: "Inter, Helvetica Neue, Arial, sans-serif",
    headingSize: 28,
    bodySize: 15,
    showPageNumber: false,
    decorationLine: false,
  },
  {
    id: "forest-moss",
    name: "森林苔藓",
    mode: "light",
    motif: "side-bar",
    accentStrategy: "block",
    palette: {
      bg: "#F6F4EE",
      surface: "#FFFFFF",
      text: "#1F2A24",
      textMuted: "#6E7D74",
      primary: "#3F6F4A",
      accent: "#B89B5E",
      accentSoft: "#E5DEC9",
      rule: "#D9D2C2",
    },
    chartPalette: {
      categorical: ["#3F6F4A", "#8FB996", "#C2D9C5", "#6E5A3C", "#B89B5E", "#4A6FA5"],
      sequential: ["#F6F4EE", "#C2D9C5", "#8FB996", "#3F6F4A", "#1F2A24"],
      diverging: ["#B89B5E", "#F6F4EE", "#3F6F4A"],
    },
    fontPair: { heading: "Source Serif Pro", body: "Source Sans Pro" },
    primary: "bg-emerald-800",
    primaryHex: "3F6F4A",
    secondary: "bg-amber-100",
    secondaryHex: "E5DEC9",
    bg: "bg-stone-50",
    bgHex: "F6F4EE",
    text: "text-emerald-950",
    textHex: "1F2A24",
    accent: "text-amber-700",
    accentHex: "B89B5E",
    chartColors: ["#3F6F4A", "#8FB996", "#C2D9C5", "#6E5A3C", "#B89B5E", "#4A6FA5"],
    fontFamily: "Source Serif Pro, Georgia, serif",
    headingSize: 26,
    bodySize: 16,
    showPageNumber: true,
    decorationLine: false,
  },
  {
    id: "coral-energy",
    name: "活力珊瑚",
    mode: "light",
    motif: "rounded-block",
    accentStrategy: "underline-num",
    palette: {
      bg: "#FFFBF7",
      surface: "#FFFFFF",
      text: "#1B1F2A",
      textMuted: "#6B6F7A",
      primary: "#FF6B6B",
      accent: "#4ECDC4",
      accentSoft: "#FFE3DC",
      rule: "#F0E5DC",
    },
    chartPalette: {
      categorical: ["#FF6B6B", "#FFB088", "#4ECDC4", "#6FC2D0", "#FFD56B", "#B47AEA"],
      sequential: ["#FFFBF7", "#FFE3DC", "#FFB088", "#FF6B6B", "#A6321F"],
      diverging: ["#FF6B6B", "#FFFBF7", "#4ECDC4"],
    },
    fontPair: { heading: "Poppins", body: "Poppins" },
    primary: "bg-rose-500",
    primaryHex: "FF6B6B",
    secondary: "bg-rose-100",
    secondaryHex: "FFE3DC",
    bg: "bg-orange-50",
    bgHex: "FFFBF7",
    text: "text-slate-900",
    textHex: "1B1F2A",
    accent: "text-teal-500",
    accentHex: "4ECDC4",
    chartColors: ["#FF6B6B", "#FFB088", "#4ECDC4", "#6FC2D0", "#FFD56B", "#B47AEA"],
    fontFamily: "Poppins, Microsoft YaHei, sans-serif",
    headingSize: 26,
    bodySize: 16,
    showPageNumber: true,
    decorationLine: false,
  },
  {
    id: "clinical-teal",
    name: "临床青",
    mode: "light",
    motif: "editorial-serif",
    accentStrategy: "heading-color",
    palette: {
      bg: "#F8FAFB",
      surface: "#FFFFFF",
      text: "#0F1B22",
      textMuted: "#516975",
      primary: "#0F766E",
      accent: "#F59E0B",
      accentSoft: "#CCFBF1",
      rule: "#D5E1E5",
    },
    chartPalette: {
      categorical: ["#0F766E", "#14B8A6", "#F59E0B", "#EF4444", "#6366F1", "#84CC16"],
      sequential: ["#F8FAFB", "#CCFBF1", "#5EEAD4", "#14B8A6", "#0F766E"],
      diverging: ["#EF4444", "#F8FAFB", "#0F766E"],
    },
    fontPair: { heading: "Cormorant Garamond", body: "Source Sans Pro" },
    primary: "bg-teal-700",
    primaryHex: "0F766E",
    secondary: "bg-teal-100",
    secondaryHex: "CCFBF1",
    bg: "bg-slate-50",
    bgHex: "F8FAFB",
    text: "text-slate-900",
    textHex: "0F1B22",
    accent: "text-amber-500",
    accentHex: "F59E0B",
    chartColors: ["#0F766E", "#14B8A6", "#F59E0B", "#EF4444", "#6366F1", "#84CC16"],
    fontFamily: "Cormorant Garamond, Georgia, serif",
    headingSize: 26,
    bodySize: 16,
    showPageNumber: true,
    decorationLine: false,
  },
];

const LEGACY_THEME_MAP: Record<string, string> = {
  "business-blue": "clinical-teal",
  "tech-dark": "midnight-executive",
  "minimal-white": "charcoal-minimal",
  "vibrant-orange": "coral-energy",
  "academic-gray": "charcoal-minimal",
  "nature-green": "forest-moss",
  "gradient-purple": "midnight-executive",
  "china-red": "warm-terracotta",
};

export function resolveThemeId(id: string): string {
  if (LEGACY_THEME_MAP[id]) return LEGACY_THEME_MAP[id];
  return id;
}

export function getTheme(id: string): PptTheme {
  const resolved = resolveThemeId(id);
  return pptThemes.find((t) => t.id === resolved) ?? pptThemes[0];
}

export const pptLanguages = [
  { code: "zh", name: "中文" },
  { code: "en", name: "English" },
];

export const slideCountOptions = [6, 8, 10, 12];

export function buildPptSystemPrompt(config: PptConfig): string {
  const lang = config.language === "zh" ? "Chinese" : "English";
  return `You are a professional presentation designer. Generate a structured presentation in ${lang} about the given topic.

Output a valid JSON object (no markdown fences, no extra text) with this exact schema:
{
  "title": "presentation title",
  "slides": [
    {
      "layout": "title" | "content" | "section" | "closing",
      "title": "slide title",
      "subtitle": "optional subtitle (for title/closing slides)",
      "bullets": ["point 1", "point 2", ...],
      "speakerNotes": "optional speaker notes"
    }
  ]
}

Rules:
- Generate exactly ${config.slideCount} slides
- First slide must use "title" layout with a compelling title and subtitle
- Last slide must use "closing" layout with a summary or thank-you message
- Use "section" layout for major topic transitions (2-3 section slides)
- Use "content" layout for detail slides with 3-5 bullet points each
- Each bullet point should be concise (under 20 words)
- Make the content informative, well-structured, and professional
- Output ONLY the JSON object, nothing else`;
}
