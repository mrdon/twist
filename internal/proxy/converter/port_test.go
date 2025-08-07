package converter

import (
	"testing"
	"time"
	"twist/internal/api"
	"twist/internal/proxy/database"
)

func TestConvertTPortToPortInfo(t *testing.T) {

	tests := []struct {
		name      string
		sectorID  int
		tport     database.TPort
		expected  *api.PortInfo
		shouldErr bool
	}{
		{
			name:     "Valid BBS port conversion",
			sectorID: 1,
			tport: database.TPort{
				Name:           "Earth Station",
				ClassIndex:     1, // BBS
				BuildTime:      5,
				Dead:           false,
				BuyProduct:     [3]bool{true, true, false}, // Buy FuelOre and Organics
				ProductPercent: [3]int{20, 30, 0},
				ProductAmount:  [3]int{0, 0, 500}, // Selling Equipment
				UpDate:         time.Date(2025, 8, 7, 12, 0, 0, 0, time.UTC),
			},
			expected: &api.PortInfo{
				SectorID:   1,
				Name:       "Earth Station",
				Class:      1,
				ClassType:  api.PortClassBBS,
				BuildTime:  5,
				Products: []api.ProductInfo{
					{Type: api.ProductTypeFuelOre, Status: api.ProductStatusBuying, Quantity: 0, Percentage: 20},
					{Type: api.ProductTypeOrganics, Status: api.ProductStatusBuying, Quantity: 0, Percentage: 30},
					{Type: api.ProductTypeEquipment, Status: api.ProductStatusSelling, Quantity: 500, Percentage: 0},
				},
				LastUpdate: time.Date(2025, 8, 7, 12, 0, 0, 0, time.UTC),
				Dead:       false,
			},
			shouldErr: false,
		},
		{
			name:     "Stardock port conversion (class 9)",
			sectorID: 100,
			tport: database.TPort{
				Name:           "Stardock Alpha",
				ClassIndex:     9, // STD
				BuildTime:      0,
				Dead:           false,
				BuyProduct:     [3]bool{false, false, false},
				ProductPercent: [3]int{0, 0, 0},
				ProductAmount:  [3]int{0, 0, 0},
				UpDate:         time.Date(2025, 8, 7, 13, 30, 0, 0, time.UTC),
			},
			expected: &api.PortInfo{
				SectorID:   100,
				Name:       "Stardock Alpha",
				Class:      9,
				ClassType:  api.PortClassSTD,
				BuildTime:  0,
				Products: []api.ProductInfo{
					{Type: api.ProductTypeFuelOre, Status: api.ProductStatusNone, Quantity: 0, Percentage: 0},
					{Type: api.ProductTypeOrganics, Status: api.ProductStatusNone, Quantity: 0, Percentage: 0},
					{Type: api.ProductTypeEquipment, Status: api.ProductStatusNone, Quantity: 0, Percentage: 0},
				},
				LastUpdate: time.Date(2025, 8, 7, 13, 30, 0, 0, time.UTC),
				Dead:       false,
			},
			shouldErr: false,
		},
		{
			name:     "Dead port conversion",
			sectorID: 50,
			tport: database.TPort{
				Name:           "Destroyed Station",
				ClassIndex:     3, // SBB
				BuildTime:      0,
				Dead:           true,
				BuyProduct:     [3]bool{false, false, false},
				ProductPercent: [3]int{0, 0, 0},
				ProductAmount:  [3]int{0, 0, 0},
				UpDate:         time.Date(2025, 8, 7, 10, 15, 0, 0, time.UTC),
			},
			expected: &api.PortInfo{
				SectorID:   50,
				Name:       "Destroyed Station",
				Class:      3,
				ClassType:  api.PortClassSBB,
				BuildTime:  0,
				Products: []api.ProductInfo{
					{Type: api.ProductTypeFuelOre, Status: api.ProductStatusNone, Quantity: 0, Percentage: 0},
					{Type: api.ProductTypeOrganics, Status: api.ProductStatusNone, Quantity: 0, Percentage: 0},
					{Type: api.ProductTypeEquipment, Status: api.ProductStatusNone, Quantity: 0, Percentage: 0},
				},
				LastUpdate: time.Date(2025, 8, 7, 10, 15, 0, 0, time.UTC),
				Dead:       true,
			},
			shouldErr: false,
		},
		{
			name:     "No port present (class 0)",
			sectorID: 25,
			tport: database.TPort{
				Name:           "",
				ClassIndex:     0, // No port
				BuildTime:      0,
				Dead:           false,
				BuyProduct:     [3]bool{false, false, false},
				ProductPercent: [3]int{0, 0, 0},
				ProductAmount:  [3]int{0, 0, 0},
				UpDate:         time.Time{},
			},
			expected:  nil,
			shouldErr: false,
		},
		{
			name:     "Complex trading port (BSS)",
			sectorID: 75,
			tport: database.TPort{
				Name:           "Trading Hub Delta",
				ClassIndex:     6, // BSS
				BuildTime:      12,
				Dead:           false,
				BuyProduct:     [3]bool{true, false, false}, // Buy only FuelOre
				ProductPercent: [3]int{85, 60, 40},
				ProductAmount:  [3]int{0, 250, 150}, // Selling Organics and Equipment
				UpDate:         time.Date(2025, 8, 7, 14, 45, 30, 0, time.UTC),
			},
			expected: &api.PortInfo{
				SectorID:   75,
				Name:       "Trading Hub Delta",
				Class:      6,
				ClassType:  api.PortClassBSS,
				BuildTime:  12,
				Products: []api.ProductInfo{
					{Type: api.ProductTypeFuelOre, Status: api.ProductStatusBuying, Quantity: 0, Percentage: 85},
					{Type: api.ProductTypeOrganics, Status: api.ProductStatusSelling, Quantity: 250, Percentage: 60},
					{Type: api.ProductTypeEquipment, Status: api.ProductStatusSelling, Quantity: 150, Percentage: 40},
				},
				LastUpdate: time.Date(2025, 8, 7, 14, 45, 30, 0, time.UTC),
				Dead:       false,
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertTPortToPortInfo(tt.sectorID, tt.tport)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil result, but got: %+v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected result, but got nil")
				return
			}

			// Verify all fields
			if result.SectorID != tt.expected.SectorID {
				t.Errorf("SectorID: expected %d, got %d", tt.expected.SectorID, result.SectorID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name: expected %s, got %s", tt.expected.Name, result.Name)
			}
			if result.Class != tt.expected.Class {
				t.Errorf("Class: expected %d, got %d", tt.expected.Class, result.Class)
			}
			if result.ClassType != tt.expected.ClassType {
				t.Errorf("ClassType: expected %v, got %v", tt.expected.ClassType, result.ClassType)
			}
			if result.BuildTime != tt.expected.BuildTime {
				t.Errorf("BuildTime: expected %d, got %d", tt.expected.BuildTime, result.BuildTime)
			}
			if result.Dead != tt.expected.Dead {
				t.Errorf("Dead: expected %t, got %t", tt.expected.Dead, result.Dead)
			}
			if !result.LastUpdate.Equal(tt.expected.LastUpdate) {
				t.Errorf("LastUpdate: expected %v, got %v", tt.expected.LastUpdate, result.LastUpdate)
			}

			// Verify products
			if len(result.Products) != len(tt.expected.Products) {
				t.Errorf("Products length: expected %d, got %d", len(tt.expected.Products), len(result.Products))
				return
			}

			for i, expectedProduct := range tt.expected.Products {
				if i >= len(result.Products) {
					t.Errorf("Missing product at index %d", i)
					continue
				}

				resultProduct := result.Products[i]
				if resultProduct.Type != expectedProduct.Type {
					t.Errorf("Product %d Type: expected %v, got %v", i, expectedProduct.Type, resultProduct.Type)
				}
				if resultProduct.Status != expectedProduct.Status {
					t.Errorf("Product %d Status: expected %v, got %v", i, expectedProduct.Status, resultProduct.Status)
				}
				if resultProduct.Quantity != expectedProduct.Quantity {
					t.Errorf("Product %d Quantity: expected %d, got %d", i, expectedProduct.Quantity, resultProduct.Quantity)
				}
				if resultProduct.Percentage != expectedProduct.Percentage {
					t.Errorf("Product %d Percentage: expected %d, got %d", i, expectedProduct.Percentage, resultProduct.Percentage)
				}
			}
		})
	}
}

func TestConvertTPortToPortInfo_IntegrationStyle(t *testing.T) {

	// Create test TPort and save to database
	testPort := database.TPort{
		Name:           "Integration Test Port",
		ClassIndex:     2, // BSB
		BuildTime:      8,
		Dead:           false,
		BuyProduct:     [3]bool{true, false, true}, // Buy FuelOre and Equipment
		ProductPercent: [3]int{75, 0, 90},
		ProductAmount:  [3]int{0, 400, 0}, // Selling Organics
		UpDate:         time.Now(),
	}

	// Test conversion
	result, err := ConvertTPortToPortInfo(123, testPort)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	// Verify basic conversion worked
	if result.SectorID != 123 {
		t.Errorf("SectorID: expected 123, got %d", result.SectorID)
	}
	if result.Name != "Integration Test Port" {
		t.Errorf("Name: expected 'Integration Test Port', got %s", result.Name)
	}
	if result.ClassType != api.PortClassBSB {
		t.Errorf("ClassType: expected PortClassBSB, got %v", result.ClassType)
	}

	// Verify product conversion logic
	expectedProducts := []api.ProductInfo{
		{Type: api.ProductTypeFuelOre, Status: api.ProductStatusBuying, Quantity: 0, Percentage: 75},
		{Type: api.ProductTypeOrganics, Status: api.ProductStatusSelling, Quantity: 400, Percentage: 0},
		{Type: api.ProductTypeEquipment, Status: api.ProductStatusBuying, Quantity: 0, Percentage: 90},
	}

	if len(result.Products) != 3 {
		t.Fatalf("Expected 3 products, got %d", len(result.Products))
	}

	for i, expected := range expectedProducts {
		actual := result.Products[i]
		if actual.Type != expected.Type || actual.Status != expected.Status ||
			actual.Quantity != expected.Quantity || actual.Percentage != expected.Percentage {
			t.Errorf("Product %d mismatch: expected %+v, got %+v", i, expected, actual)
		}
	}
}