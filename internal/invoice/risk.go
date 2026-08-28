// Rule-based triage for invoice requests: cheap validations that clear the
// obviously-fine applications for one-click batch approval, leaving humans to
// look only at the exceptions.
package invoice

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/zhulang/llm-gateway/internal/store"
)

// Chinese Unified Social Credit Code: 18 chars, digits + uppercase letters
// (excluding I, O, Z, S, V per GB 32100-2015 charset).
var usccPattern = regexp.MustCompile(`^[0-9A-HJ-NPQRTUWXY]{2}\d{6}[0-9A-HJ-NPQRTUWXY]{10}$`)

// duplicateWindowDays is the lookback window for same-title duplicate checks.
const duplicateWindowDays = 30

// EvaluateRisk runs the rule set for one request and persists the outcome.
// Returns the risk level and human-readable reasons.
func EvaluateRisk(s store.Store, req *store.InvoiceRequest) (string, []string) {
	var reasons []string

	// Rule 1: company titles need a plausible tax number.
	title, err := s.GetInvoiceTitleByID(req.TitleID)
	if err != nil {
		reasons = append(reasons, "抬头信息读取失败")
	} else if title.Type == "company" {
		tax := strings.ToUpper(strings.TrimSpace(title.TaxNumber))
		if !usccPattern.MatchString(tax) {
			reasons = append(reasons, "税号格式不符合统一社会信用代码规范")
		}
	}

	// Rule 2: amount sanity. The linked orders are verified paid at creation
	// time, so we only flag zero/negative totals (data anomaly) and unusually
	// large amounts for a second look.
	if req.TotalAmount <= 0 {
		reasons = append(reasons, "开票金额异常（≤0）")
	} else if req.TotalAmount > 50000 {
		reasons = append(reasons, fmt.Sprintf("大额开票（¥%.2f）建议人工复核", req.TotalAmount))
	}

	// Rule 3: duplicate application — same title within the window.
	if n, err := s.CountRecentInvoiceRequestsByTitle(req.TitleID, req.ID, duplicateWindowDays); err == nil && n > 0 {
		reasons = append(reasons, fmt.Sprintf("同一抬头 %d 天内已有 %d 笔在途/完成申请", duplicateWindowDays, n))
	}

	level := "auto_ok"
	if len(reasons) > 0 {
		level = "needs_review"
	}
	if err := s.SetInvoiceRequestRisk(req.ID, level, strings.Join(reasons, "；")); err == nil {
		return level, reasons
	}
	return level, reasons
}
