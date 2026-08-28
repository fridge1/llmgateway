import type PptxGenJS from "pptxgenjs";
import type { PptTheme } from "@/config/ppt-themes";
import { SLIDE } from "./spec";

// Apply per-theme background + motif decoration. Called first per slide so all subsequent
// content sits on top. Title and closing slides may override the background; chrome only
// renders motif edge elements that should remain visible.
export function renderSlideChrome(
  pres: PptxGenJS,
  s: PptxGenJS.Slide,
  theme: PptTheme,
  slideIdx: number,
  total: number,
  layout: string,
) {
  s.background = { color: theme.palette.bg.replace("#", "") };

  if (layout !== "title" && layout !== "section" && layout !== "closing") {
    drawMotif(pres, s, theme);
  }

  if (
    theme.showPageNumber &&
    slideIdx > 0 &&
    slideIdx < total - 1 &&
    layout !== "title" &&
    layout !== "section" &&
    layout !== "closing"
  ) {
    s.addText(`${String(slideIdx + 1).padStart(2, "0")} / ${String(total).padStart(2, "0")}`, {
      x: SLIDE.width - 1.4,
      y: SLIDE.height - 0.45,
      w: 1.2,
      h: 0.3,
      fontSize: 9,
      fontFace: theme.fontPair.body,
      color: theme.palette.textMuted.replace("#", ""),
      align: "right",
    });
  }
}

function drawMotif(pres: PptxGenJS, s: PptxGenJS.Slide, theme: PptTheme) {
  const accent = theme.palette.accent.replace("#", "");
  const rule = theme.palette.rule.replace("#", "");
  switch (theme.motif) {
    case "side-bar":
      s.addShape(pres.ShapeType.rect, {
        x: 0,
        y: 0,
        w: 0.32,
        h: SLIDE.height,
        fill: { color: theme.palette.primary.replace("#", "") },
        line: { type: "none" },
      });
      break;
    case "editorial-serif":
      s.addShape(pres.ShapeType.rect, {
        x: SLIDE.margin,
        y: SLIDE.height - 0.55,
        w: SLIDE.width - 2 * SLIDE.margin,
        h: 0.015,
        fill: { color: rule },
        line: { type: "none" },
      });
      break;
    case "rounded-block":
      s.addShape(pres.ShapeType.rect, {
        x: 0,
        y: SLIDE.height - 0.18,
        w: 4.5,
        h: 0.18,
        fill: { color: accent },
        line: { type: "none" },
      });
      break;
    case "split-bleed":
      s.addShape(pres.ShapeType.rect, {
        x: 0,
        y: 0,
        w: 0.7,
        h: SLIDE.height,
        fill: { color: theme.palette.accentSoft.replace("#", "") },
        line: { type: "none" },
      });
      break;
    case "card-grid":
    case "minimal-none":
      // Drawn per-layout (cards) or intentionally absent.
      break;
  }
}

// Fill a rounded card with a subtle border using the active theme.
export function paintCard(
  pres: PptxGenJS,
  s: PptxGenJS.Slide,
  theme: PptTheme,
  x: number,
  y: number,
  w: number,
  h: number,
) {
  s.addShape(pres.ShapeType.roundRect, {
    x,
    y,
    w,
    h,
    fill: { color: theme.palette.surface.replace("#", "") },
    line: {
      color: theme.palette.rule.replace("#", ""),
      width: 0.75,
    },
    rectRadius: 0.08,
  });
}

export function hex(c: string): string {
  return c.replace("#", "");
}
