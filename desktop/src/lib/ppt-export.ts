import PptxGenJS from "pptxgenjs";
import type { PresentationData } from "@/types/ppt";
import type { PptTheme } from "@/config/ppt-themes";
import { renderSlideChrome } from "./ppt-export/chrome";
import {
  renderTitle,
  renderSection,
  renderClosing,
  renderContent,
  renderIconGrid,
  renderComparison,
  renderTimeline,
  renderDataHighlight,
  renderImageText,
  renderQuote,
  renderTwoColumn,
  renderChart,
} from "./ppt-export/layouts";

export async function exportPptx(data: PresentationData, theme: PptTheme) {
  const pres = new PptxGenJS();
  pres.layout = "LAYOUT_WIDE";
  pres.author = "AI PPT Generator";
  pres.title = data.title;

  const total = data.slides.length;

  for (let slideIdx = 0; slideIdx < total; slideIdx++) {
    const slide = data.slides[slideIdx];
    const s = pres.addSlide();

    renderSlideChrome(pres, s, theme, slideIdx, total, slide.layout);

    switch (slide.layout) {
      case "title":
        renderTitle(pres, s, slide, theme);
        break;
      case "section":
        renderSection(pres, s, slide, theme);
        break;
      case "closing":
        renderClosing(pres, s, slide, theme);
        break;
      case "icon_grid":
        renderIconGrid(pres, s, slide, theme);
        break;
      case "comparison":
        renderComparison(pres, s, slide, theme);
        break;
      case "timeline":
        renderTimeline(pres, s, slide, theme);
        break;
      case "data_highlight":
        renderDataHighlight(pres, s, slide, theme);
        break;
      case "image_text":
        renderImageText(pres, s, slide, theme);
        break;
      case "quote":
        renderQuote(pres, s, slide, theme);
        break;
      case "two_column":
        renderTwoColumn(pres, s, slide, theme);
        break;
      case "chart":
        renderChart(pres, s, slide, theme);
        break;
      case "content":
      default:
        renderContent(pres, s, slide, theme);
        break;
    }

    if (slide.speakerNotes) {
      s.addNotes(slide.speakerNotes);
    }
  }

  await pres.writeFile({ fileName: `${data.title || "presentation"}.pptx` });
}
