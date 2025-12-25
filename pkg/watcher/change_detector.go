package watcher

// ChangeAnalysis describes what changed and which analysis phases need to be re-run
type ChangeAnalysis struct {
	NeedFullAnalysis bool
	NeedCompileDeps  bool
	NeedSymbolDeps   bool
	NeedBinaryDeriv  bool
	ChangedFiles     []string
}

// AnalyzeChanges determines which analysis phases need to be re-run based on what changed
func AnalyzeChanges(event ChangeEvent, workspace string) *ChangeAnalysis {
	analysis := &ChangeAnalysis{
		ChangedFiles: event.Paths,
	}

	switch event.Type {
	case ChangeTypeBuildFile:
		// BUILD file changes require full re-analysis
		// Target definitions, dependencies, or visibility changed
		analysis.NeedFullAnalysis = true
		analysis.NeedCompileDeps = true
		analysis.NeedSymbolDeps = true
		analysis.NeedBinaryDeriv = true

	case ChangeTypeDFile:
		// .d file changes mean compile dependencies changed
		// Need to re-parse .d files and update symbol deps
		analysis.NeedCompileDeps = true
		analysis.NeedSymbolDeps = true
		analysis.NeedBinaryDeriv = true

	case ChangeTypeOFile:
		// .o file changes mean symbol information changed
		// Only need to re-analyze symbols
		analysis.NeedSymbolDeps = true
		analysis.NeedBinaryDeriv = true
	}

	return analysis
}
