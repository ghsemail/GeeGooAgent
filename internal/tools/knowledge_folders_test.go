package tools

import "testing"

func TestExpandKnowledgeFolderFilter(t *testing.T) {
	got := ExpandKnowledgeFolderFilter("策略")
	if len(got) != 2 || got[0] != StrategyMaterialsFolder || got[1] != LegacyStrategyMaterialsFolder {
		t.Fatalf("materials expand = %v", got)
	}
	got = ExpandKnowledgeFolderFilter("策略认知")
	if len(got) != 2 || got[0] != StrategyArchiveFolder {
		t.Fatalf("archive expand = %v", got)
	}
}

func TestKnowledgeFolderDisplayName(t *testing.T) {
	if KnowledgeFolderDisplayName("策略") != StrategyMaterialsFolder {
		t.Fatal("expected materials display rename")
	}
	if KnowledgeFolderDisplayName("策略/子目录") != "策略资料/子目录" {
		t.Fatal("expected nested materials display rename")
	}
	if KnowledgeFolderDisplayName("策略认知") != StrategyArchiveFolder {
		t.Fatal("expected archive display rename")
	}
}

func TestNormalizeKnowledgeFolderForWrite(t *testing.T) {
	if NormalizeKnowledgeFolderForWrite("策略") != StrategyMaterialsFolder {
		t.Fatal("write normalize materials")
	}
	if NormalizeKnowledgeFolderForWrite("策略认知") != StrategyArchiveFolder {
		t.Fatal("write normalize archive")
	}
}
