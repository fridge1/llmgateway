package invoice

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

)

var companyHTTPClient = &http.Client{Timeout: 3 * time.Second}

// CompanyInfo represents basic company information.
type CompanyInfo struct {
	Name      string `json:"name"`
	TaxNumber string `json:"tax_number"`
}

// HandleCompanySearch handles GET /api/invoice/company/search?keyword=xxx
func (h *Handler) HandleCompanySearch(w http.ResponseWriter, r *http.Request) {
	keyword := strings.TrimSpace(r.URL.Query().Get("keyword"))
	if len([]rune(keyword)) < 2 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"companies": []CompanyInfo{}})
		return
	}

	companies := searchCompanies(keyword)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"companies": companies})
}

// searchCompanies queries a free public API for company info.
// Falls back to empty results on any error.
func searchCompanies(keyword string) []CompanyInfo {
	// 使用免费的企业信息查询接口
	apiURL := "https://suggest.taobao.com/sug?code=utf-8&extras=1&q=" + url.QueryEscape(keyword) + "&area=invoiceTitle"
	resp, err := companyHTTPClient.Get(apiURL)
	if err != nil {
		slog.Debug("company search failed", "error", err)
		return []CompanyInfo{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []CompanyInfo{}
	}

	// Parse taobao invoice suggestion API response
	var result struct {
		Result [][]string `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return []CompanyInfo{}
	}

	var companies []CompanyInfo
	for _, item := range result.Result {
		if len(item) >= 2 {
			companies = append(companies, CompanyInfo{
				Name:      item[0],
				TaxNumber: item[1],
			})
		}
	}
	if companies == nil {
		companies = []CompanyInfo{}
	}
	if len(companies) > 10 {
		companies = companies[:10]
	}
	return companies
}
