import { ChevronLeft, ChevronRight } from "lucide-react";
import type { PresentationData } from "@/types/ppt";
import type { PptTheme } from "@/config/ppt-themes";
import SlideChart from "./SlideChart";

interface SlidePreviewProps {
  data: PresentationData;
  theme: PptTheme;
  currentSlide: number;
  onSlideChange: (index: number) => void;
}

function TitleSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  return (
    <div className={`w-full h-full flex flex-col items-center justify-center ${theme.primary} p-8`}>
      <h1 className="text-2xl md:text-3xl font-bold text-white text-center leading-tight">
        {slide.title}
      </h1>
      {slide.subtitle && (
        <p className="mt-4 text-base md:text-lg text-white/80 text-center">{slide.subtitle}</p>
      )}
    </div>
  );
}

function SectionSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  return (
    <div className={`w-full h-full flex items-center justify-center p-8`} style={{ backgroundColor: `#${theme.accentHex}` }}>
      <h2 className="text-xl md:text-2xl font-bold text-white text-center">{slide.title}</h2>
    </div>
  );
}

function ContentSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  const body = slide.body;
  const bullets = slide.bullets ?? [];
  return (
    <div className={`w-full h-full flex flex-col ${theme.bg} p-6 md:p-8`}>
      <h2 className={`text-lg md:text-xl font-bold ${theme.text} mb-1`}>{slide.title}</h2>
      <div className="w-16 h-0.5 mb-4" style={{ backgroundColor: `#${theme.accentHex}` }} />
      <div className={`flex-1 flex ${slide.imageUrl ? 'gap-4' : ''}`}>
        <div className={slide.imageUrl ? 'flex-1 space-y-3' : 'w-full space-y-3'}>
          {body && <p className={`text-sm md:text-base ${theme.text} leading-relaxed`}>{body}</p>}
          {bullets.length > 0 && (
            <ul className="space-y-2">
              {bullets.map((bullet, i) => (
                <li key={i} className={`flex items-start gap-2 text-sm md:text-base ${theme.text}`}>
                  <span className="mt-1.5 w-1.5 h-1.5 rounded-full shrink-0" style={{ backgroundColor: `#${theme.accentHex}` }} />
                  {bullet}
                </li>
              ))}
            </ul>
          )}
        </div>
        {slide.imageUrl && (
          <div className="w-2/5 rounded-lg overflow-hidden flex items-center justify-center" style={{ backgroundColor: `#${theme.accentHex}15` }}>
            <img src={slide.imageUrl} alt={slide.title} className="w-full h-full object-cover rounded-lg" />
          </div>
        )}
      </div>
    </div>
  );
}

function IconGridSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  const items = slide.iconGrid ?? (slide.bullets ?? []).map((b) => {
    const [t, ...rest] = b.split(/[:：]/);
    return { title: t.trim(), description: rest.join(":").trim() };
  });
  const cols = items.length <= 2 ? 2 : items.length === 3 ? 3 : 2;
  return (
    <div className={`w-full h-full flex flex-col ${theme.bg} p-6 md:p-8`}>
      <h2 className={`text-lg md:text-xl font-bold ${theme.text} mb-1`}>{slide.title}</h2>
      <div className="w-16 h-0.5 mb-4" style={{ backgroundColor: `#${theme.accentHex}` }} />
      <div className={`flex-1 grid gap-3 ${cols === 3 ? 'grid-cols-3' : 'grid-cols-2'}`}>
        {items.slice(0, 6).map((it, i) => (
          <div key={i} className="rounded-lg p-3 flex flex-col gap-1" style={{ backgroundColor: `#${theme.accentHex}10` }}>
            <div className="flex items-center gap-2">
              <span className="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold shrink-0" style={{ backgroundColor: `#${theme.accentHex}`, color: "#fff" }}>
                {i + 1}
              </span>
              <p className={`text-sm md:text-base font-semibold ${theme.text}`}>{it.title}</p>
            </div>
            {it.description && (
              <p className={`text-xs md:text-sm ${theme.text} opacity-75 leading-relaxed`}>{it.description}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function ClosingSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  return (
    <div className={`w-full h-full flex flex-col items-center justify-center ${theme.primary} p-8`}>
      <h2 className="text-xl md:text-2xl font-bold text-white text-center">{slide.title}</h2>
      {slide.subtitle && (
        <p className="mt-3 text-sm md:text-base text-white/70 text-center">{slide.subtitle}</p>
      )}
    </div>
  );
}

function ComparisonSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  const structured = slide.comparison;
  let headers: [string, string] = ["", ""];
  let rows: [string, string][] = [];
  if (structured) {
    headers = structured.headers;
    rows = structured.rows;
  } else {
    const items = slide.bullets || [];
    const half = Math.ceil(items.length / 2);
    const left = items.slice(0, half);
    const right = items.slice(half);
    const n = Math.max(left.length, right.length);
    rows = Array.from({ length: n }, (_, i) => [left[i] ?? "", right[i] ?? ""]);
  }
  return (
    <div className={`w-full h-full flex flex-col ${theme.bg} p-6 md:p-8`}>
      <h2 className={`text-lg md:text-xl font-bold ${theme.text} mb-1`}>{slide.title}</h2>
      <div className="w-16 h-0.5 mb-4" style={{ backgroundColor: `#${theme.accentHex}` }} />
      <div className="flex-1 grid grid-cols-2 gap-4">
        <div className="space-y-2">
          {headers[0] && (
            <p className={`text-sm font-semibold pb-1 border-b ${theme.text}`} style={{ borderColor: `#${theme.accentHex}40` }}>{headers[0]}</p>
          )}
          {rows.map(([l], i) => l && (
            <div key={i} className={`flex items-start gap-2 text-sm ${theme.text}`}>
              <span className="mt-1.5 w-1.5 h-1.5 rounded-full shrink-0" style={{ backgroundColor: `#${theme.accentHex}` }} />
              {l}
            </div>
          ))}
        </div>
        <div className="space-y-2">
          {headers[1] && (
            <p className={`text-sm font-semibold pb-1 border-b ${theme.text}`} style={{ borderColor: `#${theme.accentHex}40` }}>{headers[1]}</p>
          )}
          {rows.map(([, r], i) => r && (
            <div key={i} className={`flex items-start gap-2 text-sm ${theme.text}`}>
              <span className="mt-1.5 w-1.5 h-1.5 rounded-full shrink-0 bg-gray-400" />
              {r}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function TimelineSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  const items = slide.timelineItems ?? (slide.bullets ?? []).map((b) => ({ time: "", title: b, description: "" }));
  return (
    <div className={`w-full h-full flex flex-col ${theme.bg} p-6 md:p-8`}>
      <h2 className={`text-lg md:text-xl font-bold ${theme.text} mb-1`}>{slide.title}</h2>
      <div className="w-16 h-0.5 mb-4" style={{ backgroundColor: `#${theme.accentHex}` }} />
      <div className="flex-1 flex items-center">
        <div className="w-full flex items-start gap-1">
          {items.map((it, i) => (
            <div key={i} className="flex-1 flex flex-col items-center text-center px-1">
              {it.time && (
                <span className="text-[10px] md:text-xs font-semibold mb-1" style={{ color: `#${theme.accentHex}` }}>{it.time}</span>
              )}
              <div className="w-3 h-3 rounded-full mb-2" style={{ backgroundColor: `#${theme.accentHex}` }} />
              <p className={`text-xs md:text-sm font-semibold ${theme.text}`}>{it.title}</p>
              {it.description && (
                <p className={`text-[10px] md:text-xs ${theme.text} opacity-70 mt-1`}>{it.description}</p>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function DataHighlightSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  if (slide.chartData) {
    const chartColors = theme.chartColors || ["#2563EB", "#F97316", "#10B981", "#8B5CF6", "#EF4444", "#06B6D4"];
    return (
      <div className={`w-full h-full flex flex-col ${theme.bg} p-6 md:p-8`}>
        <h2 className={`text-lg md:text-xl font-bold ${theme.text} mb-2`}>{slide.title}</h2>
        <div className="flex-1 min-h-0">
          <SlideChart data={slide.chartData} colors={chartColors} />
        </div>
      </div>
    );
  }

  const stats = slide.stats ?? (slide.bullets ?? []).map((b) => ({ value: b, label: "" }));
  return (
    <div className={`w-full h-full flex flex-col ${theme.bg} p-6 md:p-8`}>
      <h2 className={`text-lg md:text-xl font-bold ${theme.text} mb-4`}>{slide.title}</h2>
      <div className={`flex-1 grid gap-4 items-center ${stats.length <= 2 ? 'grid-cols-2' : 'grid-cols-3'}`}>
        {stats.slice(0, 3).map((s, i) => (
          <div key={i} className="text-center p-4 rounded-lg" style={{ backgroundColor: `#${theme.accentHex}15` }}>
            <p className="text-2xl md:text-4xl font-bold leading-none" style={{ color: `#${theme.accentHex}` }}>
              {s.value}
              {s.unit && <span className="text-base md:text-xl ml-0.5">{s.unit}</span>}
            </p>
            {s.label && <p className={`mt-2 text-xs md:text-sm ${theme.text} opacity-75`}>{s.label}</p>}
          </div>
        ))}
      </div>
      {stats.length > 3 && (
        <div className="mt-2 space-y-1">
          {stats.slice(3).map((s, i) => (
            <p key={i} className={`text-sm ${theme.text} opacity-70`}>
              {s.value}{s.unit ?? ""} {s.label ? `· ${s.label}` : ""}
            </p>
          ))}
        </div>
      )}
    </div>
  );
}

function ImageTextSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  return (
    <div className={`w-full h-full flex ${theme.bg} p-6 md:p-8`}>
      <div className="flex-1 flex flex-col justify-center pr-4">
        <h2 className={`text-lg md:text-xl font-bold ${theme.text} mb-3`}>{slide.title}</h2>
        <ul className="space-y-2">
          {slide.bullets?.map((b, i) => (
            <li key={i} className={`flex items-start gap-2 text-sm ${theme.text}`}>
              <span className="mt-1.5 w-1.5 h-1.5 rounded-full shrink-0" style={{ backgroundColor: `#${theme.accentHex}` }} />
              {b}
            </li>
          ))}
        </ul>
      </div>
      <div className="w-2/5 rounded-lg flex items-center justify-center overflow-hidden" style={{ backgroundColor: `#${theme.accentHex}15` }}>
        {slide.imageUrl ? (
          <img src={slide.imageUrl} alt={slide.title} className="w-full h-full object-cover rounded-lg" />
        ) : (
          <span className={`text-sm ${theme.text} opacity-40`}>图片区域</span>
        )}
      </div>
    </div>
  );
}

function QuoteSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  const quote = slide.quote?.text ?? slide.bullets?.[0] ?? slide.subtitle ?? "";
  const attribution = slide.quote?.attribution ?? slide.bullets?.[1] ?? "";
  return (
    <div className={`w-full h-full flex flex-col items-center justify-center ${theme.bg} p-8 md:p-12`}>
      <span className="text-5xl md:text-6xl opacity-20" style={{ color: `#${theme.accentHex}` }}>&ldquo;</span>
      <p className={`text-lg md:text-2xl font-medium ${theme.text} text-center max-w-[80%] -mt-4`}>
        {quote}
      </p>
      {attribution && (
        <p className={`mt-4 text-sm ${theme.text} opacity-60`}>— {attribution}</p>
      )}
    </div>
  );
}

function TwoColumnSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  const items = slide.bullets || [];
  const half = Math.ceil(items.length / 2);
  const left = items.slice(0, half);
  const right = items.slice(half);
  return (
    <div className={`w-full h-full flex flex-col ${theme.bg} p-6 md:p-8`}>
      <h2 className={`text-lg md:text-xl font-bold ${theme.text} mb-1`}>{slide.title}</h2>
      <div className="w-16 h-0.5 mb-4" style={{ backgroundColor: `#${theme.accentHex}` }} />
      <div className="flex-1 grid grid-cols-2 gap-6">
        <div className="space-y-2">
          {left.map((b, i) => (
            <div key={i} className={`flex items-start gap-2 text-sm ${theme.text}`}>
              <span className="mt-1.5 w-1.5 h-1.5 rounded-full shrink-0" style={{ backgroundColor: `#${theme.accentHex}` }} />
              {b}
            </div>
          ))}
        </div>
        <div className="space-y-2 border-l pl-6" style={{ borderColor: `#${theme.accentHex}30` }}>
          {right.map((b, i) => (
            <div key={i} className={`flex items-start gap-2 text-sm ${theme.text}`}>
              <span className="mt-1.5 w-1.5 h-1.5 rounded-full shrink-0" style={{ backgroundColor: `#${theme.accentHex}` }} />
              {b}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function ChartSlide({ slide, theme }: { slide: PresentationData["slides"][0]; theme: PptTheme }) {
  const chartColors = theme.chartColors || ["#2563EB", "#F97316", "#10B981", "#8B5CF6", "#EF4444", "#06B6D4"];
  return (
    <div className={`w-full h-full flex flex-col ${theme.bg} p-6 md:p-8`}>
      <h2 className={`text-lg md:text-xl font-bold ${theme.text} mb-2`}>{slide.title}</h2>
      {slide.chartData ? (
        <div className="flex-1 min-h-0">
          <SlideChart data={slide.chartData} colors={chartColors} />
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center">
          <p className={`text-sm ${theme.text} opacity-40`}>图表区域</p>
        </div>
      )}
      {slide.subtitle && (
        <p className={`mt-2 text-sm ${theme.text} opacity-70 text-center`}>{slide.subtitle}</p>
      )}
    </div>
  );
}

const layoutComponents: Record<string, typeof TitleSlide> = {
  title: TitleSlide,
  section: SectionSlide,
  content: ContentSlide,
  closing: ClosingSlide,
  comparison: ComparisonSlide,
  timeline: TimelineSlide,
  data_highlight: DataHighlightSlide,
  image_text: ImageTextSlide,
  icon_grid: IconGridSlide,
  quote: QuoteSlide,
  two_column: TwoColumnSlide,
  chart: ChartSlide,
};

export default function SlidePreview({ data, theme, currentSlide, onSlideChange }: SlidePreviewProps) {
  const slide = data.slides[currentSlide];
  if (!slide) return null;

  const Layout = layoutComponents[slide.layout] || ContentSlide;

  return (
    <div className="flex flex-col items-center gap-4 w-full">
      {/* Main preview */}
      <div className="relative w-full max-w-4xl">
        <div className="relative w-full" style={{ paddingBottom: "56.25%" }}>
          <div className="absolute inset-0 rounded-lg overflow-hidden shadow-lg border border-border">
            <Layout slide={slide} theme={theme} />
          </div>
        </div>
        {/* Nav arrows */}
        {currentSlide > 0 && (
          <button
            onClick={() => onSlideChange(currentSlide - 1)}
            className="absolute left-2 top-1/2 -translate-y-1/2 p-1.5 rounded-full bg-black/40 text-white hover:bg-black/60 transition-colors cursor-pointer"
          >
            <ChevronLeft size={20} />
          </button>
        )}
        {currentSlide < data.slides.length - 1 && (
          <button
            onClick={() => onSlideChange(currentSlide + 1)}
            className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 rounded-full bg-black/40 text-white hover:bg-black/60 transition-colors cursor-pointer"
          >
            <ChevronRight size={20} />
          </button>
        )}
        {/* Slide counter */}
        <div className="absolute bottom-2 right-3 px-2 py-0.5 rounded bg-black/50 text-white text-xs">
          {currentSlide + 1} / {data.slides.length}
        </div>
      </div>

      {/* Thumbnail strip */}
      <div className="flex gap-2 overflow-x-auto pb-2 max-w-4xl w-full px-1">
        {data.slides.map((s, i) => {
          const ThumbLayout = layoutComponents[s.layout] || ContentSlide;
          return (
            <button
              key={i}
              onClick={() => onSlideChange(i)}
              className={`shrink-0 w-24 h-[54px] rounded border-2 overflow-hidden transition-all cursor-pointer ${
                i === currentSlide ? "border-primary ring-1 ring-primary" : "border-border hover:border-muted-foreground"
              }`}
            >
              <div className="w-full h-full transform scale-[0.15] origin-top-left" style={{ width: "160px", height: "90px" }}>
                <div style={{ width: "160px", height: "90px", fontSize: "4px" }}>
                  <ThumbLayout slide={s} theme={theme} />
                </div>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
