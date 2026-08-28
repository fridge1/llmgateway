package ppt

// BriefDocument is the output of Agent 1 (Brief Analyst).
type BriefDocument struct {
	PresentationType  string   `json:"presentation_type"`
	Audience          Audience `json:"audience"`
	NarrativeGoal     string   `json:"narrative_goal"`
	TimeMinutes       int      `json:"time_constraint_minutes"`
	SlideCountRange   [2]int   `json:"slide_count_range"`
	Tone              string   `json:"tone"`
	KeyMessages       []string `json:"key_messages"`
	MandatorySections []string `json:"mandatory_sections"`
	DataRequirements  []string `json:"data_requirements,omitempty"`
	SuggestedVisuals  []string `json:"suggested_visuals,omitempty"`
}

type Audience struct {
	Role           string   `json:"role"`
	KnowledgeLevel string   `json:"knowledge_level"`
	DecisionFocus  []string `json:"decision_focus"`
}

// StoryArc is the output of Agent 2 (Content Strategist).
type StoryArc struct {
	NarrativePattern string       `json:"narrative_pattern"`
	Slides           []StorySlide `json:"slides"`
}

type StorySlide struct {
	Position         int      `json:"position"`
	Role             string   `json:"role"`
	CoreMessage      string   `json:"core_message"`
	EmotionalBeat    string   `json:"emotional_beat"`
	TransitionLogic  string   `json:"transition_logic"`
	SpeakerNotes     string   `json:"speaker_notes,omitempty"`
	DataPoints       []string `json:"data_points,omitempty"`
	VisualSuggestion string   `json:"visual_suggestion,omitempty"`
}

// SlideBlueprint is the output of Agent 3 (Info Architect).
type SlideBlueprint struct {
	SlideID            int            `json:"slide_id"`
	ContentType        string         `json:"content_type"`
	LayoutRationale    string         `json:"layout_rationale,omitempty"`
	Elements           []SlideElement `json:"elements"`
	InformationDensity float64        `json:"information_density"`
	VisualEmphasis     string         `json:"visual_emphasis"`
	SpeakerNotes       string         `json:"speaker_notes,omitempty"`
}

type ChartSeries struct {
	Label  string    `json:"label"`
	Values []float64 `json:"values"`
}

type SlideElement struct {
	Type      string    `json:"type"`
	Content   string    `json:"content,omitempty"`
	Hierarchy int       `json:"hierarchy,omitempty"`
	Columns   []string  `json:"columns,omitempty"`
	Rows      int       `json:"rows,omitempty"`
	Items     []string  `json:"items,omitempty"`
	// StructuredItems holds objects for icon_grid / timeline_items / comparison_table rows.
	// Keys vary by element type (see prompts.go ENCODING CONVENTIONS).
	StructuredItems []map[string]string `json:"structured_items,omitempty"`
	ChartType       string              `json:"chart_type,omitempty"`
	Labels          []string            `json:"labels,omitempty"`
	Values          []float64           `json:"values,omitempty"`
	Series          []ChartSeries       `json:"series,omitempty"`
	// stat_number auxiliary fields.
	Unit  string `json:"unit,omitempty"`
	Label string `json:"label,omitempty"`
	// quote auxiliary field.
	Attribution string `json:"attribution,omitempty"`
}

// SlideBlueprintSet is the full output of Agent 3.
type SlideBlueprintSet struct {
	Blueprints []SlideBlueprint `json:"blueprints"`
}

// ImagePlan is the output of Agent 4 (Visual Designer).
type ImagePlan struct {
	Images []ImageSpec `json:"images"`
}

type ImageSpec struct {
	SlideIndex int    `json:"slide_index"`
	Prompt     string `json:"prompt"`
	Style      string `json:"style"`
	ImageSlot  string `json:"image_slot,omitempty"`
}

// PptTask mirrors the ppt_tasks DB row for use within the ppt package.
type PptTask struct {
	ID              int                `json:"id"`
	UserID          string             `json:"user_id"`
	Status          string             `json:"status"`
	Phase           string             `json:"phase"`
	Topic           string             `json:"topic"`
	SlideCount      int                `json:"slide_count"`
	Language        string             `json:"language"`
	Theme           string             `json:"theme"`
	Audience        string             `json:"audience"`
	Tone            string             `json:"tone"`
	Purpose         string             `json:"purpose"`
	Model           string             `json:"model"`
	OutlineOnly     bool               `json:"outline_only"`
	GenerateImages  bool               `json:"generate_images"`
	ContextText     string             `json:"context_text,omitempty"`
	BriefDocument   *BriefDocument     `json:"brief_document,omitempty"`
	StoryArc        *StoryArc          `json:"story_arc,omitempty"`
	SlideBlueprints *SlideBlueprintSet `json:"slide_blueprints,omitempty"`
	PresentationJSON interface{}       `json:"presentation_json,omitempty"`
	TotalTokens     int                `json:"total_tokens"`
	Cost            float64            `json:"cost"`
	ErrorMessage    string             `json:"error_message,omitempty"`
	CreatedAt       string             `json:"created_at"`
	StartedAt       *string            `json:"started_at,omitempty"`
	CompletedAt     *string            `json:"completed_at,omitempty"`
}
