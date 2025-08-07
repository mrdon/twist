

package parsing

import (
	"testing"
)

func TestSectorAndPortParsing(t *testing.T) {
	AssertTuiApiCalls(t, "sector_and_port_data.txt", []string{
		"OnCurrentSectorChanged({\"number\":2142,\"nav_haz\":0,\"has_traders\":0,\"constellation\":\"uncharted space\",\"beacon\":\"\",\"warps\":[8247,18964],\"has_port\":true})",
	})
}
