package tools

import "strings"

const (
	// StrategyMaterialsFolder holds reference strategy PDFs / external docs uploaded to WeKnora.
	StrategyMaterialsFolder = "策略资料"
	// LegacyStrategyMaterialsFolder is the pre-rename folder path (read fallback).
	LegacyStrategyMaterialsFolder = "策略"

	// StrategyArchiveFolder holds Agent-generated strategy profile markdown.
	StrategyArchiveFolder = "策略档案"
	// SignalDiagnoseFolder holds Agent-generated signal diagnosis reports.
	SignalDiagnoseFolder = "信号诊断"
	// LegacyStrategyArchiveFolder is the pre-rename archive folder (read fallback).
	LegacyStrategyArchiveFolder = "策略认知"
)

// ExpandKnowledgeFolderFilter returns folder paths to query (includes legacy aliases).
func ExpandKnowledgeFolderFilter(folder string) []string {
	folder = strings.TrimSpace(folder)
	switch folder {
	case "":
		return []string{""}
	case StrategyMaterialsFolder, LegacyStrategyMaterialsFolder:
		return []string{StrategyMaterialsFolder, LegacyStrategyMaterialsFolder}
	case StrategyArchiveFolder, LegacyStrategyArchiveFolder:
		return []string{StrategyArchiveFolder, LegacyStrategyArchiveFolder}
	case SignalDiagnoseFolder:
		return []string{SignalDiagnoseFolder}
	default:
		return []string{folder}
	}
}

// NormalizeKnowledgeFolderForWrite maps legacy folder names to canonical paths for new writes.
func NormalizeKnowledgeFolderForWrite(folder string) string {
	folder = strings.TrimSpace(folder)
	switch folder {
	case LegacyStrategyMaterialsFolder, StrategyMaterialsFolder:
		return StrategyMaterialsFolder
	case LegacyStrategyArchiveFolder, StrategyArchiveFolder:
		return StrategyArchiveFolder
	default:
		return folder
	}
}

// KnowledgeFolderDisplayName returns user-facing folder label for API / UI.
func KnowledgeFolderDisplayName(folderPath string) string {
	folderPath = strings.Trim(strings.ReplaceAll(folderPath, "\\", "/"), "/")
	if folderPath == "" {
		return ""
	}
	if folderPath == LegacyStrategyMaterialsFolder {
		return StrategyMaterialsFolder
	}
	if folderPath == LegacyStrategyArchiveFolder {
		return StrategyArchiveFolder
	}
	parts := strings.Split(folderPath, "/")
	if parts[0] == LegacyStrategyMaterialsFolder {
		parts[0] = StrategyMaterialsFolder
		return strings.Join(parts, "/")
	}
	if parts[0] == LegacyStrategyArchiveFolder {
		parts[0] = StrategyArchiveFolder
		return strings.Join(parts, "/")
	}
	return folderPath
}
