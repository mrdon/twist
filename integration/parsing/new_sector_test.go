//go:build integration

package parsing

import (
	"testing"
)

func TestNewSectorParsing(t *testing.T) {
	AssertTuiApiCalls(t, "new_sector_data.txt", []string{
		"OnCurrentSectorChanged({\"number\":2142,\"nav_haz\":0,\"has_traders\":0,\"constellation\":\"uncharted space\",\"beacon\":\"\",\"warps\":[8247,18964]})",
		"OnCurrentSectorChanged({\"number\":8247,\"nav_haz\":0,\"has_traders\":0,\"constellation\":\"uncharted space (unexplored)\",\"beacon\":\"\",\"warps\":[2142,13975,16563,16589]})",
		"OnCurrentSectorChanged({\"number\":13975,\"nav_haz\":0,\"has_traders\":0,\"constellation\":\"uncharted space (unexplored)\",\"beacon\":\"\",\"warps\":[8247]})",
	})
}