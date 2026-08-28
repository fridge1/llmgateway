// Shared layout spec referenced by both the .pptx exporter and the React preview.
// Coordinates assume a 13.33 × 7.5 inch (LAYOUT_WIDE) slide.

export const SLIDE = {
  width: 13.33,
  height: 7.5,
  margin: 0.8,
  contentY: 0.6,
  headingHeight: 0.85,
} as const;

export const TYPE = {
  titleSize: 54,
  sectionSize: 44,
  headingSize: 28,
  subheadingSize: 18,
  bodySize: 16,
  smallSize: 13,
  captionSize: 11,
  bigStatSize: 60,
  quoteSize: 32,
} as const;

export const COLOR = {
  // Transparency knobs used by motifs and image fallbacks.
  scrim: 35,
  cardSoft: 90,
  fallbackShape: 65,
} as const;

export type LayoutId =
  | "title"
  | "content"
  | "section"
  | "closing"
  | "comparison"
  | "timeline"
  | "data_highlight"
  | "image_text"
  | "icon_grid"
  | "quote"
  | "two_column"
  | "chart";
