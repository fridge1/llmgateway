package ppt

import (
	"testing"
)

func TestBlueprintsToPresentation_PreservesIconGrid(t *testing.T) {
	bp := &SlideBlueprintSet{
		Blueprints: []SlideBlueprint{
			{
				SlideID:         3,
				ContentType:     "bullet_list", // simulate the real-world bug: agent said bullet_list but emitted icon_grid
				LayoutRationale: "3-4 parallel concepts",
				Elements: []SlideElement{
					{Type: "heading", Content: "AI医疗四大核心场景"},
					{Type: "icon_grid", StructuredItems: []map[string]string{
						{"icon": "scan", "title": "影像诊断", "description": "最成熟的ROI最快场景"},
						{"icon": "flask", "title": "药物研发", "description": "增长最迅猛"},
						{"icon": "chat", "title": "智能问诊", "description": "渗透最广"},
						{"icon": "robot", "title": "手术机器人", "description": "单价最高"},
					}},
				},
			},
		},
	}
	brief := &BriefDocument{KeyMessages: []string{"AI医疗落地加速"}}

	out := blueprintsToPresentation(bp, brief, "clinical-teal")
	slides, _ := out["slides"].([]map[string]interface{})
	if len(slides) != 1 {
		t.Fatalf("expected 1 slide, got %d", len(slides))
	}
	s := slides[0]
	if layout := s["layout"]; layout != "icon_grid" {
		t.Errorf("expected resolved layout icon_grid, got %v", layout)
	}
	grid, ok := s["iconGrid"].([]map[string]string)
	if !ok || len(grid) != 4 {
		t.Fatalf("iconGrid missing or wrong length: %#v", s["iconGrid"])
	}
	if grid[0]["title"] != "影像诊断" {
		t.Errorf("first icon_grid title wrong: %v", grid[0]["title"])
	}
	if s["layoutRationale"] != "3-4 parallel concepts" {
		t.Errorf("layoutRationale not preserved")
	}
}

func TestBlueprintsToPresentation_PreservesStatNumber(t *testing.T) {
	bp := &SlideBlueprintSet{
		Blueprints: []SlideBlueprint{
			{
				SlideID:     4,
				ContentType: "data_highlight",
				Elements: []SlideElement{
					{Type: "heading", Content: "关键指标"},
					{Type: "stat_number", Content: "94", Unit: "%", Label: "肺结节敏感度"},
					{Type: "stat_number", Content: "40", Unit: "%", Label: "阅片时间节省"},
				},
			},
		},
	}
	brief := &BriefDocument{KeyMessages: []string{"x"}}
	out := blueprintsToPresentation(bp, brief, "clinical-teal")
	slides := out["slides"].([]map[string]interface{})
	stats, ok := slides[0]["stats"].([]map[string]interface{})
	if !ok || len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %#v", slides[0]["stats"])
	}
	if stats[0]["value"] != "94" || stats[0]["unit"] != "%" || stats[0]["label"] != "肺结节敏感度" {
		t.Errorf("stat[0] wrong: %#v", stats[0])
	}
}

func TestBlueprintsToPresentation_MultiSeriesChart(t *testing.T) {
	bp := &SlideBlueprintSet{
		Blueprints: []SlideBlueprint{
			{
				SlideID:     1,
				ContentType: "chart",
				Elements: []SlideElement{
					{Type: "heading", Content: "同比对比"},
					{Type: "chart_data", ChartType: "bar", Labels: []string{"Q1", "Q2", "Q3"},
						Series: []ChartSeries{
							{Label: "2024", Values: []float64{10, 20, 30}},
							{Label: "2025", Values: []float64{15, 25, 35}},
						}},
				},
			},
		},
	}
	brief := &BriefDocument{KeyMessages: []string{"x"}}
	out := blueprintsToPresentation(bp, brief, "clinical-teal")
	slides := out["slides"].([]map[string]interface{})
	chart, ok := slides[0]["chartData"].(map[string]interface{})
	if !ok {
		t.Fatalf("chartData missing")
	}
	datasets, _ := chart["datasets"].([]map[string]interface{})
	if len(datasets) != 2 {
		t.Fatalf("expected 2 datasets, got %d", len(datasets))
	}
	if datasets[0]["label"] != "2024" {
		t.Errorf("first series label wrong: %v", datasets[0]["label"])
	}
}

func TestBlueprintsToPresentation_Comparison(t *testing.T) {
	bp := &SlideBlueprintSet{
		Blueprints: []SlideBlueprint{
			{
				SlideID:     2,
				ContentType: "comparison_matrix",
				Elements: []SlideElement{
					{Type: "heading", Content: "X vs Y"},
					{Type: "comparison_table",
						Columns: []string{"传统", "AI"},
						StructuredItems: []map[string]string{
							{"left": "人工阅片10分钟", "right": "AI辅助2分钟"},
							{"left": "漏诊率12%", "right": "漏诊率4%"},
						}},
				},
			},
		},
	}
	brief := &BriefDocument{KeyMessages: []string{"x"}}
	out := blueprintsToPresentation(bp, brief, "clinical-teal")
	slides := out["slides"].([]map[string]interface{})
	cmp, ok := slides[0]["comparison"].(map[string]interface{})
	if !ok {
		t.Fatalf("comparison missing")
	}
	rows, _ := cmp["rows"].([][2]string)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0][0] != "人工阅片10分钟" || rows[0][1] != "AI辅助2分钟" {
		t.Errorf("row 0 wrong: %v", rows[0])
	}
}

func TestBlueprintsToPresentation_PipeFallback(t *testing.T) {
	// Old LLM output using pipe-delimited Items should still parse.
	bp := &SlideBlueprintSet{
		Blueprints: []SlideBlueprint{
			{
				SlideID:     5,
				ContentType: "timeline",
				Elements: []SlideElement{
					{Type: "heading", Content: "时间线"},
					{Type: "timeline_items", Items: []string{
						"2024-Q1|启动|试点部署",
						"2024-Q2|放量|服务10家医院",
					}},
				},
			},
		},
	}
	brief := &BriefDocument{KeyMessages: []string{"x"}}
	out := blueprintsToPresentation(bp, brief, "clinical-teal")
	slides := out["slides"].([]map[string]interface{})
	items, ok := slides[0]["timelineItems"].([]map[string]string)
	if !ok || len(items) != 2 {
		t.Fatalf("timelineItems missing: %#v", slides[0]["timelineItems"])
	}
	if items[0]["time"] != "2024-Q1" || items[0]["title"] != "启动" {
		t.Errorf("parsed timeline wrong: %#v", items[0])
	}
}

func TestStylePrefixPrompt(t *testing.T) {
	cases := []struct {
		style string
		want  string
	}{
		{"photograph", "Professional editorial photograph, a scene. No text, no captions, no logos."},
		{"illustration", "Modern flat vector illustration, a scene. No text, no captions, no logos."},
		{"", "a scene. No text, no captions, no logos."},
	}
	for _, c := range cases {
		got := stylePrefixPrompt(ImageSpec{Prompt: "a scene", Style: c.style})
		if got != c.want {
			t.Errorf("style=%q: got %q, want %q", c.style, got, c.want)
		}
	}
}
