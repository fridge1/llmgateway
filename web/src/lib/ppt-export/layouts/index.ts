import type PptxGenJS from "pptxgenjs";
import type { ChartData, SlideData } from "@/types/ppt";
import type { PptTheme } from "@/config/ppt-themes";
import { SLIDE, TYPE } from "../spec";
import { hex, paintCard } from "../chrome";
import { placeImage } from "../image";

type Pres = PptxGenJS;
type Slide = PptxGenJS.Slide;

function addChart(pres: Pres, s: Slide, chartData: ChartData, theme: PptTheme, x: number, y: number, w: number, h: number) {
  const chartTypeMap: Record<string, PptxGenJS.CHART_NAME> = {
    bar: pres.ChartType.bar,
    line: pres.ChartType.line,
    pie: pres.ChartType.pie,
    doughnut: pres.ChartType.doughnut,
  };
  const t = chartTypeMap[chartData.type] || pres.ChartType.bar;
  const data = chartData.datasets.map((ds, i) => ({
    name: ds.label || `Series ${i + 1}`,
    labels: chartData.labels,
    values: ds.values,
  }));
  s.addChart(t, data, {
    x, y, w, h,
    showTitle: false,
    showLegend: chartData.datasets.length > 1,
    legendPos: "b",
    chartColors: theme.chartPalette.categorical.map((c) => c.replace("#", "")),
  });
}

// Heading helper used by every text-driven layout. Bold + theme heading font + accent
// strategy applied. Returns the y position content should start below.
function heading(s: Slide, theme: PptTheme, title: string): number {
  const useAccentColor = theme.accentStrategy === "heading-color";
  s.addText(title, {
    x: SLIDE.margin,
    y: SLIDE.contentY,
    w: SLIDE.width - 2 * SLIDE.margin,
    h: SLIDE.headingHeight,
    fontSize: TYPE.headingSize,
    fontFace: theme.fontPair.heading,
    bold: true,
    color: hex(useAccentColor ? theme.palette.accent : theme.palette.text),
  });
  return SLIDE.contentY + SLIDE.headingHeight + 0.25;
}

// ---------------- title ----------------

export function renderTitle(_pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  s.background = { color: hex(theme.palette.primary) };
  s.addText(slide.title, {
    x: 0.8, y: 2.6, w: SLIDE.width - 1.6, h: 1.6,
    fontSize: TYPE.titleSize,
    fontFace: theme.fontPair.heading,
    bold: true,
    color: theme.mode === "dark" ? "FFFFFF" : "FFFFFF",
    align: "center",
  });
  if (slide.subtitle) {
    s.addText(slide.subtitle, {
      x: 0.8, y: 4.4, w: SLIDE.width - 1.6, h: 1.0,
      fontSize: TYPE.subheadingSize + 4,
      fontFace: theme.fontPair.body,
      color: "E0E6F0",
      align: "center",
    });
  }
}

// ---------------- section ----------------

export function renderSection(_pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  // Two-tone vertical split: 40% primary, 60% surface.
  const splitX = SLIDE.width * 0.4;
  s.background = { color: hex(theme.palette.surface) };
  s.addShape(_pres.ShapeType.rect, {
    x: 0, y: 0, w: splitX, h: SLIDE.height,
    fill: { color: hex(theme.palette.primary) },
    line: { type: "none" },
  });
  s.addText(slide.title, {
    x: splitX + 0.7, y: 2.8, w: SLIDE.width - splitX - 1.4, h: 1.6,
    fontSize: TYPE.sectionSize,
    fontFace: theme.fontPair.heading,
    bold: true,
    color: hex(theme.palette.text),
  });
  if (slide.subtitle) {
    s.addText(slide.subtitle, {
      x: splitX + 0.7, y: 4.5, w: SLIDE.width - splitX - 1.4, h: 0.8,
      fontSize: TYPE.subheadingSize,
      fontFace: theme.fontPair.body,
      color: hex(theme.palette.textMuted),
    });
  }
}

// ---------------- closing ----------------

export function renderClosing(pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  if (slide.imageUrl) {
    placeImage(pres, s, slide, theme, "background_overlay");
  } else {
    s.background = { color: hex(theme.palette.primary) };
  }
  s.addText(slide.title, {
    x: 0.8, y: 2.4, w: SLIDE.width - 1.6, h: 1.4,
    fontSize: TYPE.titleSize - 4,
    fontFace: theme.fontPair.heading,
    bold: true,
    color: "FFFFFF",
    align: "center",
  });
  if (slide.subtitle || slide.body) {
    s.addText(slide.subtitle ?? slide.body ?? "", {
      x: 1.4, y: 4.0, w: SLIDE.width - 2.8, h: 1.4,
      fontSize: TYPE.subheadingSize + 2,
      fontFace: theme.fontPair.body,
      color: "E5E7EB",
      align: "center",
      valign: "top",
    });
  }
}

// ---------------- content (body + bullets) ----------------

export function renderContent(_pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  const bodyStartY = heading(s, theme, slide.title);
  const hasImage = !!slide.imageUrl;
  const textW = hasImage ? 7.4 : SLIDE.width - 2 * SLIDE.margin;

  let cursor = bodyStartY;
  if (slide.body) {
    s.addText(slide.body, {
      x: SLIDE.margin, y: cursor, w: textW, h: 1.4,
      fontSize: TYPE.bodySize + 1,
      fontFace: theme.fontPair.body,
      color: hex(theme.palette.text),
      valign: "top",
    });
    cursor += 1.4;
  }
  if (slide.bullets?.length) {
    s.addText(
      slide.bullets.map((b) => ({
        text: b,
        options: {
          fontSize: TYPE.bodySize,
          fontFace: theme.fontPair.body,
          color: hex(theme.palette.text),
          bullet: { code: "2022" as const },
          paraSpaceAfter: 10,
        },
      })),
      { x: SLIDE.margin, y: cursor, w: textW, h: SLIDE.height - cursor - 0.6, valign: "top" },
    );
  }

  if (hasImage) {
    placeImage(_pres, s, slide, theme, "card");
  }
}

// ---------------- icon_grid ----------------

export function renderIconGrid(pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  const startY = heading(s, theme, slide.title);
  const items = slide.iconGrid ?? [];
  const visible = items.slice(0, 6);
  const cols = visible.length <= 2 ? 2 : visible.length === 3 ? 3 : 2;
  const rows = Math.ceil(visible.length / cols);
  const gutter = 0.3;
  const cellW = (SLIDE.width - 2 * SLIDE.margin - (cols - 1) * gutter) / cols;
  const cellH = (SLIDE.height - startY - 0.6 - (rows - 1) * gutter) / rows;

  visible.forEach((item, i) => {
    const r = Math.floor(i / cols);
    const c = i % cols;
    const x = SLIDE.margin + c * (cellW + gutter);
    const y = startY + r * (cellH + gutter);
    paintCard(pres, s, theme, x, y, cellW, cellH);
    s.addShape(pres.ShapeType.ellipse, {
      x: x + 0.3, y: y + 0.3, w: 0.55, h: 0.55,
      fill: { color: hex(theme.palette.accent) },
      line: { type: "none" },
    });
    s.addText(`${i + 1}`, {
      x: x + 0.3, y: y + 0.3, w: 0.55, h: 0.55,
      fontSize: 16, fontFace: theme.fontPair.body, bold: true,
      color: "FFFFFF", align: "center", valign: "middle",
    });
    s.addText(item.title, {
      x: x + 1.05, y: y + 0.3, w: cellW - 1.3, h: 0.6,
      fontSize: TYPE.bodySize + 2, fontFace: theme.fontPair.heading, bold: true,
      color: hex(theme.palette.text), valign: "middle",
    });
    if (item.description) {
      s.addText(item.description, {
        x: x + 0.3, y: y + 1.0, w: cellW - 0.6, h: cellH - 1.2,
        fontSize: TYPE.smallSize, fontFace: theme.fontPair.body,
        color: hex(theme.palette.textMuted), valign: "top", paraSpaceAfter: 4,
      });
    }
  });
}

// ---------------- comparison ----------------

export function renderComparison(_pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  const startY = heading(s, theme, slide.title);
  const headers = slide.comparison?.headers ?? ["", ""];
  let rows: [string, string][] = [];
  if (slide.comparison) {
    rows = slide.comparison.rows;
  } else {
    const items = slide.bullets ?? [];
    const half = Math.ceil(items.length / 2);
    const left = items.slice(0, half);
    const right = items.slice(half);
    const n = Math.max(left.length, right.length);
    rows = Array.from({ length: n }, (_, i) => [left[i] ?? "", right[i] ?? ""]);
  }

  const colW = (SLIDE.width - 2 * SLIDE.margin - 0.4) / 2;
  const headerH = 0.55;

  s.addShape(_pres.ShapeType.rect, {
    x: SLIDE.margin, y: startY, w: colW, h: headerH,
    fill: { color: hex(theme.palette.accentSoft) },
    line: { type: "none" },
  });
  s.addShape(_pres.ShapeType.rect, {
    x: SLIDE.margin + colW + 0.4, y: startY, w: colW, h: headerH,
    fill: { color: hex(theme.palette.accentSoft) },
    line: { type: "none" },
  });
  if (headers[0]) {
    s.addText(headers[0], {
      x: SLIDE.margin + 0.2, y: startY, w: colW - 0.4, h: headerH,
      fontSize: TYPE.bodySize + 1, fontFace: theme.fontPair.heading, bold: true,
      color: hex(theme.palette.accent), valign: "middle",
    });
  }
  if (headers[1]) {
    s.addText(headers[1], {
      x: SLIDE.margin + colW + 0.6, y: startY, w: colW - 0.4, h: headerH,
      fontSize: TYPE.bodySize + 1, fontFace: theme.fontPair.heading, bold: true,
      color: hex(theme.palette.accent), valign: "middle",
    });
  }

  const rowH = Math.min(0.65, (SLIDE.height - startY - headerH - 0.7) / Math.max(rows.length, 1));
  const rowsY = startY + headerH;
  rows.slice(0, 8).forEach((row, i) => {
    const y = rowsY + i * rowH;
    if (i % 2 === 1) {
      s.addShape(_pres.ShapeType.rect, {
        x: SLIDE.margin, y, w: SLIDE.width - 2 * SLIDE.margin, h: rowH,
        fill: { color: hex(theme.palette.surface) },
        line: { type: "none" },
      });
    }
    if (row[0]) {
      s.addText(row[0], {
        x: SLIDE.margin + 0.2, y, w: colW - 0.4, h: rowH,
        fontSize: TYPE.bodySize - 1, fontFace: theme.fontPair.body,
        color: hex(theme.palette.text), valign: "middle",
      });
    }
    if (row[1]) {
      s.addText(row[1], {
        x: SLIDE.margin + colW + 0.6, y, w: colW - 0.4, h: rowH,
        fontSize: TYPE.bodySize - 1, fontFace: theme.fontPair.body,
        color: hex(theme.palette.text), valign: "middle",
      });
    }
  });
}

// ---------------- timeline (horizontal, alternating captions) ----------------

export function renderTimeline(pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  heading(s, theme, slide.title);
  const items = slide.timelineItems
    ?? (slide.bullets ?? []).map((b) => ({ time: "", title: b, description: "" }));
  const n = items.length;
  if (n === 0) return;

  const lineY = SLIDE.height / 2 + 0.2;
  const startX = SLIDE.margin + 0.3;
  const endX = SLIDE.width - SLIDE.margin - 0.3;
  const step = (endX - startX) / Math.max(n - 1, 1);

  s.addShape(pres.ShapeType.rect, {
    x: startX, y: lineY - 0.02, w: endX - startX, h: 0.04,
    fill: { color: hex(theme.palette.rule) },
    line: { type: "none" },
  });

  items.forEach((it, i) => {
    const cx = n === 1 ? (startX + endX) / 2 : startX + i * step;
    const above = i % 2 === 0;
    s.addShape(pres.ShapeType.ellipse, {
      x: cx - 0.13, y: lineY - 0.13, w: 0.26, h: 0.26,
      fill: { color: hex(theme.palette.accent) },
      line: { type: "none" },
    });
    if (it.time) {
      s.addText(it.time, {
        x: cx - 1.1, y: above ? lineY - 1.65 : lineY + 0.45,
        w: 2.2, h: 0.3,
        fontSize: TYPE.captionSize, fontFace: theme.fontPair.body, bold: true,
        color: hex(theme.palette.accent), align: "center",
      });
    }
    s.addText(it.title, {
      x: cx - 1.1, y: above ? lineY - 1.3 : lineY + 0.8,
      w: 2.2, h: 0.5,
      fontSize: TYPE.bodySize - 1, fontFace: theme.fontPair.heading, bold: true,
      color: hex(theme.palette.text), align: "center", valign: "top",
    });
    if (it.description) {
      s.addText(it.description, {
        x: cx - 1.2, y: above ? lineY - 0.7 : lineY + 1.4,
        w: 2.4, h: 0.6,
        fontSize: TYPE.smallSize, fontFace: theme.fontPair.body,
        color: hex(theme.palette.textMuted), align: "center", valign: "top",
      });
    }
  });
}

// ---------------- data_highlight (stats vs chart-with-narrative) ----------------

export function renderDataHighlight(pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  heading(s, theme, slide.title);

  if (slide.chartData) {
    // Left half: chart. Right half: takeaway + bullets.
    addChart(pres, s, slide.chartData, theme, 0.6, 1.7, 6.6, 5.0);
    if (slide.body || slide.bullets?.length || slide.subtitle) {
      const right = slide.body || slide.subtitle || slide.bullets?.[0] || "";
      if (right) {
        s.addText(right, {
          x: 7.6, y: 1.7, w: 5.0, h: 1.6,
          fontSize: 22, fontFace: theme.fontPair.heading, bold: true,
          color: hex(theme.palette.text), valign: "top",
        });
      }
      if (slide.bullets?.length && slide.body) {
        s.addText(slide.bullets.map((b) => ({
          text: b,
          options: {
            fontSize: TYPE.bodySize - 1, fontFace: theme.fontPair.body,
            color: hex(theme.palette.text), bullet: { code: "2022" as const }, paraSpaceAfter: 6,
          },
        })), { x: 7.6, y: 3.6, w: 5.0, h: 3.0, valign: "top" });
      }
    }
    return;
  }

  const stats = slide.stats ?? (slide.bullets ?? []).map((b) => ({ value: b, unit: "", label: "" }));
  const visible = stats.slice(0, 3);
  if (visible.length === 0) return;
  const gutter = 0.4;
  const cardW = (SLIDE.width - 2 * SLIDE.margin - (visible.length - 1) * gutter) / visible.length;
  const cardY = 2.0;
  const cardH = 4.2;
  visible.forEach((stat, i) => {
    const x = SLIDE.margin + i * (cardW + gutter);
    paintCard(pres, s, theme, x, cardY, cardW, cardH);
    const valueText = stat.value;
    const unit = stat.unit ?? "";
    s.addText(valueText, {
      x: x + 0.2, y: cardY + 0.6, w: cardW - 0.4, h: 1.8,
      fontSize: TYPE.bigStatSize, fontFace: theme.fontPair.heading, bold: true,
      color: hex(theme.palette.accent), align: "center", valign: "middle",
    });
    if (unit) {
      s.addText(unit, {
        x: x + 0.2, y: cardY + 2.4, w: cardW - 0.4, h: 0.4,
        fontSize: 22, fontFace: theme.fontPair.body, bold: true,
        color: hex(theme.palette.accent), align: "center", valign: "top",
      });
    }
    if (stat.label) {
      s.addText(stat.label, {
        x: x + 0.3, y: cardY + 3.0, w: cardW - 0.6, h: 1.0,
        fontSize: TYPE.smallSize + 1, fontFace: theme.fontPair.body,
        color: hex(theme.palette.textMuted), align: "center", valign: "top",
      });
    }
  });
}

// ---------------- image_text (full-bleed image + copy) ----------------

export function renderImageText(pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  const slot: "full_bleed_left" | "full_bleed_right" = slide.imageSlot === "full_bleed_left"
    ? "full_bleed_left"
    : "full_bleed_right";
  placeImage(pres, s, slide, theme, slot);
  const copyX = slot === "full_bleed_left" ? SLIDE.width / 2 + 0.6 : 0.6;
  const copyW = SLIDE.width / 2 - 1.2;
  s.addText(slide.title, {
    x: copyX, y: 1.2, w: copyW, h: 1.2,
    fontSize: TYPE.headingSize + 2, fontFace: theme.fontPair.heading, bold: true,
    color: hex(theme.palette.text), valign: "top",
  });
  let cursor = 2.6;
  if (slide.body) {
    s.addText(slide.body, {
      x: copyX, y: cursor, w: copyW, h: 1.6,
      fontSize: TYPE.bodySize, fontFace: theme.fontPair.body,
      color: hex(theme.palette.text), valign: "top",
    });
    cursor += 1.6;
  }
  if (slide.bullets?.length) {
    s.addText(slide.bullets.map((b) => ({
      text: b,
      options: {
        fontSize: TYPE.bodySize - 1, fontFace: theme.fontPair.body,
        color: hex(theme.palette.text), bullet: { code: "2022" as const }, paraSpaceAfter: 6,
      },
    })), { x: copyX, y: cursor, w: copyW, h: SLIDE.height - cursor - 0.5, valign: "top" });
  }
}

// ---------------- quote ----------------

export function renderQuote(pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  s.addShape(pres.ShapeType.rect, {
    x: 0.8, y: 1.2, w: 0.06, h: 5.0,
    fill: { color: hex(theme.palette.accent) },
    line: { type: "none" },
  });
  const quote = slide.quote?.text ?? slide.bullets?.[0] ?? slide.subtitle ?? "";
  const attribution = slide.quote?.attribution ?? slide.bullets?.[1] ?? "";
  s.addText(quote, {
    x: 1.6, y: 1.6, w: SLIDE.width - 2.4, h: 4.0,
    fontSize: TYPE.quoteSize, fontFace: theme.fontPair.heading, italic: true,
    color: hex(theme.palette.text), valign: "middle",
  });
  if (attribution) {
    s.addText(`— ${attribution}`, {
      x: 1.6, y: 5.8, w: SLIDE.width - 2.4, h: 0.6,
      fontSize: TYPE.bodySize, fontFace: theme.fontPair.body,
      color: hex(theme.palette.textMuted),
    });
  }
}

// ---------------- two_column ----------------

export function renderTwoColumn(pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  const startY = heading(s, theme, slide.title);
  const colW = (SLIDE.width - 2 * SLIDE.margin - 0.6) / 2;
  const items = slide.bullets ?? [];
  const half = Math.ceil(items.length / 2);
  const left = items.slice(0, half);
  const right = items.slice(half);
  if (left.length) {
    s.addText(left.map((b) => ({
      text: b,
      options: {
        fontSize: TYPE.bodySize - 1, fontFace: theme.fontPair.body,
        color: hex(theme.palette.text), bullet: { code: "2022" as const }, paraSpaceAfter: 8,
      },
    })), { x: SLIDE.margin, y: startY, w: colW, h: SLIDE.height - startY - 0.6, valign: "top" });
  }
  s.addShape(pres.ShapeType.rect, {
    x: SLIDE.margin + colW + 0.3 - 0.01, y: startY, w: 0.02, h: SLIDE.height - startY - 0.6,
    fill: { color: hex(theme.palette.rule) },
    line: { type: "none" },
  });
  if (right.length) {
    s.addText(right.map((b) => ({
      text: b,
      options: {
        fontSize: TYPE.bodySize - 1, fontFace: theme.fontPair.body,
        color: hex(theme.palette.text), bullet: { code: "2022" as const }, paraSpaceAfter: 8,
      },
    })), { x: SLIDE.margin + colW + 0.6, y: startY, w: colW, h: SLIDE.height - startY - 0.6, valign: "top" });
  }
}

// ---------------- chart ----------------

export function renderChart(pres: Pres, s: Slide, slide: SlideData, theme: PptTheme) {
  heading(s, theme, slide.title);
  if (slide.chartData) {
    addChart(pres, s, slide.chartData, theme, SLIDE.margin, 1.7, SLIDE.width - 2 * SLIDE.margin, 5.0);
  }
  if (slide.subtitle) {
    s.addText(slide.subtitle, {
      x: SLIDE.margin, y: SLIDE.height - 0.7, w: SLIDE.width - 2 * SLIDE.margin, h: 0.4,
      fontSize: TYPE.captionSize, fontFace: theme.fontPair.body,
      color: hex(theme.palette.textMuted), align: "center",
    });
  }
}
