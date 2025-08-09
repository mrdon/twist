package parsing

import (
	"fmt"
	"testing"
	"twist/internal/proxy/database"
)

func TestSectorReportParsing(t *testing.T) {
	// Create test parser
	parser, mockAPI, db := CreateTestParser(t)
	defer db.CloseDatabase()

	// Process the real-world sector report data
	for _, chunk := range ParseDataChunks("sector_report_data.txt") {
		parser.ProcessInBound(string(chunk))
	}

	// Force completion of parsing
	parser.Finalize()

	// Verify that sectors were saved to database
	testCases := []struct {
		sectorNum    int
		expectedWarps []int
	}{
		{1, []int{2, 3, 4, 5, 6, 7}},
		{2, []int{1, 3, 7, 8, 9, 10}},
		{3, []int{1, 2, 4, 8008, 12952}},
		{8, []int{2, 7, 13429}},
		{2142, []int{8247, 18964}},
		{18964, []int{2142, 10424}},
		{19987, []int{2173, 2278, 10866}},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Sector_%d", tc.sectorNum), func(t *testing.T) {
			// Load sector from database
			sector, err := db.LoadSector(tc.sectorNum)
			if err != nil {
				t.Fatalf("Failed to load sector %d: %v", tc.sectorNum, err)
			}

			// Verify warp data was parsed and saved
			actualWarps := []int{}
			for _, warp := range sector.Warp {
				if warp > 0 {
					actualWarps = append(actualWarps, warp)
				}
			}

			if len(actualWarps) != len(tc.expectedWarps) {
				t.Errorf("Sector %d: expected %d warps, got %d. Expected: %v, Got: %v", 
					tc.sectorNum, len(tc.expectedWarps), len(actualWarps), tc.expectedWarps, actualWarps)
				return
			}

			// Verify each warp
			for i, expectedWarp := range tc.expectedWarps {
				if i >= len(actualWarps) || actualWarps[i] != expectedWarp {
					t.Errorf("Sector %d: warp %d expected %d, got %d", tc.sectorNum, i, expectedWarp, actualWarps[i])
				}
			}

			// Verify the sector was marked as updated
			if sector.UpDate.IsZero() {
				t.Errorf("Sector %d: UpDate should not be zero", tc.sectorNum)
			}

			// Verify exploration status
			expectedExplored := database.EtCalc // CIM data marks sectors as calculated
			if sector.Explored != expectedExplored {
				t.Errorf("Sector %d: expected Explored=%d, got %d", tc.sectorNum, expectedExplored, sector.Explored)
			}
		})
	}

	// Verify TUI API was NOT called for CIM data (CIM data is background processing)
	calls := mockAPI.GetCalls()
	if len(calls) > 0 {
		t.Errorf("Expected no TUI API calls for CIM data processing, got %d calls: %v", len(calls), calls)
	}
}