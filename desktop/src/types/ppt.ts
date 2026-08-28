export type SlideLayout =
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

export interface ChartData {
  type: "bar" | "line" | "pie" | "doughnut";
  title?: string;
  labels: string[];
  datasets: { label: string; values: number[] }[];
}

export interface IconGridItem {
  icon?: string;
  title: string;
  description?: string;
}

export interface ComparisonTable {
  headers: [string, string];
  rows: [string, string][];
}

export interface TimelineItem {
  time: string;
  title: string;
  description?: string;
}

export interface StatNumber {
  value: string;
  unit?: string;
  label?: string;
}

export interface QuoteContent {
  text: string;
  attribution?: string;
}

export type ImageStyle = "photograph" | "illustration" | "abstract" | "diagram";
export type ImageSlot =
  | "full_bleed_left"
  | "full_bleed_right"
  | "background_overlay"
  | "card"
  | "icon_circle";

export interface SlideData {
  layout: SlideLayout;
  title: string;
  subtitle?: string;
  body?: string;
  bullets?: string[];
  iconGrid?: IconGridItem[];
  comparison?: ComparisonTable;
  timelineItems?: TimelineItem[];
  stats?: StatNumber[];
  quote?: QuoteContent;
  chartData?: ChartData;
  imageUrl?: string;
  imageStyle?: ImageStyle;
  imageSlot?: ImageSlot;
  layoutRationale?: string;
  speakerNotes?: string;
}

export interface PresentationData {
  title: string;
  slides: SlideData[];
}

export interface PptConfig {
  topic: string;
  slideCount: number;
  theme: string;
  language: string;
}

// Async task types
export interface StoryArcSlide {
  position: number;
  role: string;
  core_message: string;
  emotional_beat: string;
  transition_logic: string;
  speaker_notes?: string;
  data_points?: string[];
  visual_suggestion?: string;
}

export interface StoryArc {
  narrative_pattern: string;
  slides: StoryArcSlide[];
}

export interface PptTask {
  id: number;
  user_id: string;
  status: "pending" | "processing" | "completed" | "failed" | "outline_ready";
  phase: string;
  topic: string;
  slide_count: number;
  language: string;
  theme: string;
  audience: string;
  tone: string;
  purpose: string;
  model: string;
  outline_only?: boolean;
  story_arc?: StoryArc;
  presentation_json?: PresentationData;
  total_tokens: number;
  cost: number;
  error_message?: string;
  created_at: string;
  started_at?: string;
  completed_at?: string;
}

export interface SubmitPptTaskRequest {
  model: string;
  topic: string;
  slide_count: number;
  language: string;
  theme: string;
  audience: string;
  tone: string;
  purpose: string;
  outline_only?: boolean;
  generate_images?: boolean;
  context_text?: string;
}

export interface SubmitPptTaskResponse {
  id: number;
  status: string;
}
