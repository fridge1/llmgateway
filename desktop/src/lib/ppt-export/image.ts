import type PptxGenJS from "pptxgenjs";
import type { PptTheme } from "@/config/ppt-themes";
import type { ImageSlot, SlideData } from "@/types/ppt";
import { SLIDE } from "./spec";
import { hex } from "./chrome";

// Map a layout to its default image slot when the slide didn't specify one.
export function defaultImageSlot(layout: string): ImageSlot {
  switch (layout) {
    case "image_text":
      return "full_bleed_right";
    case "section":
    case "closing":
      return "background_overlay";
    case "data_highlight":
    case "content":
      return "card";
    default:
      return "card";
  }
}

// Draw the slide image (or a tasteful fallback) for layouts that admit one.
export function placeImage(
  pres: PptxGenJS,
  s: PptxGenJS.Slide,
  slide: SlideData,
  theme: PptTheme,
  fallbackSlot?: ImageSlot,
) {
  const slot = slide.imageSlot ?? fallbackSlot ?? defaultImageSlot(slide.layout);
  if (!slide.imageUrl) {
    drawImageFallback(pres, s, theme, slot);
    return;
  }
  const url = slide.imageUrl;
  switch (slot) {
    case "full_bleed_left":
      s.addImage({ path: url, x: 0, y: 0, w: SLIDE.width / 2, h: SLIDE.height, sizing: { type: "cover", w: SLIDE.width / 2, h: SLIDE.height } });
      break;
    case "full_bleed_right":
      s.addImage({ path: url, x: SLIDE.width / 2, y: 0, w: SLIDE.width / 2, h: SLIDE.height, sizing: { type: "cover", w: SLIDE.width / 2, h: SLIDE.height } });
      break;
    case "background_overlay":
      s.addImage({ path: url, x: 0, y: 0, w: SLIDE.width, h: SLIDE.height, sizing: { type: "cover", w: SLIDE.width, h: SLIDE.height } });
      s.addShape(pres.ShapeType.rect, {
        x: 0,
        y: 0,
        w: SLIDE.width,
        h: SLIDE.height,
        fill: { color: hex(theme.palette.primary), transparency: 35 },
        line: { type: "none" },
      });
      break;
    case "card":
      s.addImage({ path: url, x: SLIDE.width - 3.0, y: 0.5, w: 2.3, h: 1.5 });
      break;
    case "icon_circle":
      s.addImage({ path: url, x: SLIDE.margin, y: 1.4, w: 1.0, h: 1.0, rounding: true });
      break;
  }
}

// When no image URL is available, draw motif-flavored shapes in the same slot region —
// never leave a visible empty box. For "card"/"icon_circle" we fall through silently.
export function drawImageFallback(
  pres: PptxGenJS,
  s: PptxGenJS.Slide,
  theme: PptTheme,
  slot: ImageSlot,
) {
  const accent = hex(theme.palette.accent);
  const accentSoft = hex(theme.palette.accentSoft);

  if (slot === "full_bleed_right") {
    s.addShape(pres.ShapeType.ellipse, {
      x: SLIDE.width / 2 + 1.0, y: 1.2, w: 4.0, h: 4.0,
      fill: { color: accent, transparency: 65 },
      line: { type: "none" },
    });
    s.addShape(pres.ShapeType.roundRect, {
      x: SLIDE.width / 2 + 2.5, y: 3.0, w: 3.0, h: 3.0,
      fill: { color: accentSoft, transparency: 25 },
      line: { type: "none" },
      rectRadius: 0.4,
    });
    s.addShape(pres.ShapeType.rect, {
      x: SLIDE.width / 2 + 0.6, y: 5.5, w: 1.2, h: 1.2,
      fill: { color: accent, transparency: 80 },
      line: { type: "none" },
    });
    return;
  }
  if (slot === "full_bleed_left") {
    s.addShape(pres.ShapeType.ellipse, {
      x: 1.0, y: 1.2, w: 4.0, h: 4.0,
      fill: { color: accent, transparency: 65 },
      line: { type: "none" },
    });
    s.addShape(pres.ShapeType.roundRect, {
      x: 2.5, y: 3.0, w: 3.0, h: 3.0,
      fill: { color: accentSoft, transparency: 25 },
      line: { type: "none" },
      rectRadius: 0.4,
    });
    s.addShape(pres.ShapeType.rect, {
      x: 0.6, y: 5.5, w: 1.2, h: 1.2,
      fill: { color: accent, transparency: 80 },
      line: { type: "none" },
    });
    return;
  }
  if (slot === "background_overlay") {
    // Tinted plate keeps the closing slide from looking bare.
    s.addShape(pres.ShapeType.rect, {
      x: 0, y: 0, w: SLIDE.width, h: SLIDE.height,
      fill: { color: hex(theme.palette.primary) },
      line: { type: "none" },
    });
  }
  // card / icon_circle: silently no-op; layouts that *require* an image must check imageUrl themselves.
}
