package climgen

func recordCategoryTradeDiagnostic(
	diagnostics *MultimodalTradeDiagnostics,
	category string,
	update func(*MultimodalTradeCategoryDiagnostics),
) {
	if diagnostics == nil || category == "" || update == nil {
		return
	}
	if diagnostics.ByCategory == nil {
		diagnostics.ByCategory = map[string]MultimodalTradeCategoryDiagnostics{}
	}
	entry := diagnostics.ByCategory[category]
	entry.Category = category
	update(&entry)
	diagnostics.ByCategory[category] = entry
}
