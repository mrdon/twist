

package parsing

import (
	"testing"
)

func TestMultipleSectorsParsing(t *testing.T) {
	AssertTuiApiCalls(t, "multiple_sectors_data.txt", []string{
		"OnCurrentSectorChanged({\"number\":2142,\"nav_haz\":0,\"has_traders\":0,\"constellation\":\"uncharted space\",\"beacon\":\"\",\"warps\":[8247,18964],\"has_port\":true})",
		"OnCurrentSectorChanged({\"number\":2142,\"nav_haz\":0,\"has_traders\":0,\"constellation\":\"uncharted space\",\"beacon\":\"\",\"warps\":[8247,18964],\"has_port\":true})",
		"OnCurrentSectorChanged({\"number\":18964,\"nav_haz\":0,\"has_traders\":0,\"constellation\":\"uncharted space\",\"beacon\":\"\",\"warps\":[2142,10424]})",
		"OnCurrentSectorChanged({\"number\":2142,\"nav_haz\":0,\"has_traders\":0,\"constellation\":\"uncharted space\",\"beacon\":\"\",\"warps\":[8247,18964],\"has_port\":true})",
	})
}