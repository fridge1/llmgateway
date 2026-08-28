// Package export holds shared spreadsheet-export helpers so the admin, user
// and tenant transaction exports produce identical columns.
package export

import (
	"fmt"
	"net/http"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/zhulang/llm-gateway/internal/store"
)

// WriteTransactionsXLSX streams transactions as an Excel file. The column set
// is the single source of truth shared by admin/user/tenant exports.
func WriteTransactionsXLSX(w http.ResponseWriter, filenamePrefix string, transactions []store.Transaction) error {
	f := excelize.NewFile()
	sheet := "Sheet1"

	headers := []string{"时间", "类型", "模型", "输入Token", "输出Token", "缓存命中", "缓存写入", "金额(元)"}
	for i, head := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, head)
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E2EFDA"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "H1", headerStyle)

	cellName := func(col, row int) string {
		name, _ := excelize.CoordinatesToCellName(col, row)
		return name
	}

	for row, tx := range transactions {
		rIdx := row + 2
		f.SetCellValue(sheet, cellName(1, rIdx), tx.CreatedAt.Format("2006-01-02 15:04:05"))
		f.SetCellValue(sheet, cellName(2, rIdx), tx.Type)
		if tx.Model != nil {
			f.SetCellValue(sheet, cellName(3, rIdx), *tx.Model)
		}
		if tx.PromptTokens != nil {
			f.SetCellValue(sheet, cellName(4, rIdx), *tx.PromptTokens)
		}
		if tx.CompletionTokens != nil {
			f.SetCellValue(sheet, cellName(5, rIdx), *tx.CompletionTokens)
		}
		if tx.CacheReadTokens != nil {
			f.SetCellValue(sheet, cellName(6, rIdx), *tx.CacheReadTokens)
		}
		if tx.CacheCreationTokens != nil {
			f.SetCellValue(sheet, cellName(7, rIdx), *tx.CacheCreationTokens)
		}
		f.SetCellValue(sheet, cellName(8, rIdx), fmt.Sprintf("%.4f", tx.Amount))
	}

	for i := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, 16)
	}

	filename := fmt.Sprintf("%s_%s.xlsx", filenamePrefix, time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return f.Write(w)
}
