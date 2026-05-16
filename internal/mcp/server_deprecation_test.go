package mcp

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestLegacyToolsCarryDeprecatedMarker(t *testing.T) {
	legacy := []mcp.Tool{
		searchTool(),
		addResourceTool(),
		getRoadmapTool(),
		triageTool(),
		listDomainsTool(),
		listProjectsTool(),
		createDomainTool(),
		createSubareaTool(),
		createProjectTool(),
		linkProjectTool(),
		toggleProjectTool(),
		discoverProjectTool(),
		importProjectTool(),
		createSessionTool(),
		createLearningTool(),
		listSessionsTool(),
		listLearningsTool(),
		linkResourceTool(),
		listPersonTool(),
		syncObsidianTool(),
	}
	for _, tool := range legacy {
		t.Run(tool.Name, func(t *testing.T) {
			if !strings.HasPrefix(tool.Description, "[DEPRECATED]") {
				t.Fatalf("expected %s description to start with [DEPRECATED], got %q", tool.Name, tool.Description)
			}
		})
	}
}

func TestBridgeToolsAreNotDeprecated(t *testing.T) {
	bridge := []mcp.Tool{
		readFromEngramTool(),
		syncToObsidianTool(),
		searchVaultTool(),
		createOrUpdateNoteTool(),
		getKnowledgeContextTool(),
	}
	for _, tool := range bridge {
		t.Run(tool.Name, func(t *testing.T) {
			if strings.Contains(tool.Description, "[DEPRECATED]") {
				t.Fatalf("bridge tool %s should NOT be marked deprecated, got %q", tool.Name, tool.Description)
			}
		})
	}
}
