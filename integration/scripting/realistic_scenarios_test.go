//go:build integration

package scripting

import (
	"testing"
)

// TestTWXLoginSequence_RealIntegration tests a realistic login script pattern with real components
func TestTWXLoginSequence_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# Realistic TWX login sequence based on 1_Login.ts
echo "Starting login sequence..."

# Set up variables like real TWX scripts
setvar $loginName "TestUser"
setvar $password "TestPass123"
setvar $game "A"

# Simulate login steps
echo "Waiting for login prompt..."
send $loginName
echo "Sent username: " $loginName

echo "Waiting for game selection..."
send $game
echo "Selected game: " $game

echo "Entering password..."
send $password
echo "Login sequence completed"
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	// Verify commands sent to game
	expectedCommands := []string{"TestUser", "A", "TestPass123"}
	tester.AssertCommands(result, expectedCommands)
	
	// Verify proper output sequence
	expectedOutputs := []string{
		"Starting login sequence...",
		"Waiting for login prompt...",
		"Sent username: TestUser",
		"Waiting for game selection...",
		"Selected game: A", 
		"Entering password...",
		"Login sequence completed",
	}
	tester.AssertOutput(result, expectedOutputs)
}

// TestSectorNavigation_RealIntegration tests navigation patterns from real scripts with real VM
func TestSectorNavigation_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# Navigation script based on 1_Move.ts patterns
echo "Starting sector navigation..."

# Set destination
setvar $targetSector 1500
setvar $currentSector 1000

echo "Current sector: " $currentSector
echo "Target sector: " $targetSector

# Calculate if we need to move
isequal $targetSector $currentSector $isEqual
if $isEqual <> 1
  echo "Moving to sector " $targetSector
  send "m"
  send $targetSector
  echo "Movement command sent"
else
  echo "Already at target sector"
end

echo "Navigation completed"
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	// Verify movement commands
	expectedCommands := []string{"m", "1500"}
	tester.AssertCommands(result, expectedCommands)
	
	// Verify navigation logic
	tester.AssertOutputContains(result, "Current sector: 1000")
	tester.AssertOutputContains(result, "Target sector: 1500")
	tester.AssertOutputContains(result, "Moving to sector 1500")
	tester.AssertOutputContains(result, "Movement command sent")
}

// TestPortTrading_RealIntegration tests trading patterns from real scripts with real array persistence
func TestPortTrading_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# Port trading script based on 1_Trade.ts patterns
echo "Checking port status..."

# Set up product tracking arrays like real scripts
array $products 3
setarrayelement $products 0 "Fuel Ore"
setarrayelement $products 1 "Organics"
setarrayelement $products 2 "Equipment"

# Set up prices array
array $prices 3
setarrayelement $prices 0 "10"
setarrayelement $prices 1 "15"
setarrayelement $prices 2 "25"

# Check each product
$i := 0
while $i < 3
  getarrayelement $products $i $product
  getarrayelement $prices $i $price
  echo "Product: " $product " Price: " $price
  
  # Simulate buying logic - convert price to number for comparison
  add $price 0 $priceNum
  if $priceNum < 20
    echo "Buying " $product " at good price: " $price
    send "b"
    send $product
  else
    echo "Price too high for " $product
  end
  
  add $i 1 $i
end

echo "Trading analysis complete"
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	// Verify trading commands sent
	expectedCommands := []string{"b", "Fuel Ore", "b", "Organics"}
	tester.AssertCommands(result, expectedCommands)
	
	// Verify trading logic output
	tester.AssertOutputContains(result, "Product: Fuel Ore Price: 10")
	tester.AssertOutputContains(result, "Buying Fuel Ore at good price: 10")
	tester.AssertOutputContains(result, "Price too high for Equipment")
}

// TestStringProcessing_RealIntegration tests string handling from real scripts with real VM
func TestStringProcessing_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# String processing like in 1_Move.ts
echo "Processing game text..."

# Simulate parsing game output
$currentLine := "Sector  : 1500     Warps to Sectors : 1499, 1501"
echo "Parsing line: " $currentLine

# Extract sector number (word 3)
getword $currentLine $sector 3
echo "Current sector: " $sector

# Extract warp information  
cuttext $currentLine $warpText 20 25
echo "Warp text: " $warpText

# Validate results
isequal $sector "1500" $sectorMatch
if $sectorMatch = 1
  echo "Sector parsing successful"
else
  echo "Sector parsing failed"
end
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	// Verify string processing results
	expectedOutputs := []string{
		"Processing game text...",
		"Parsing line: Sector  : 1500     Warps to Sectors : 1499, 1501",
		"Current sector: 1500",
		"Warp text: Warps",
		"Sector parsing successful",
	}
	tester.AssertOutput(result, expectedOutputs)
}

// TestWeightingSystem_RealIntegration tests decision logic from real scripts with real calculation
func TestWeightingSystem_RealIntegration(t *testing.T) {
	tester := NewIntegrationScriptTester(t)
	
	script := `
# Weighting system like in 1_Trade.ts
echo "Calculating sector weights..."

# Set up sector data arrays
array $sectors 3
array $densities 3
array $weights 3

setarrayelement $sectors 0 "1000"
setarrayelement $sectors 1 "1001" 
setarrayelement $sectors 2 "1002"

setarrayelement $densities 0 "100"
setarrayelement $densities 1 "50"
setarrayelement $densities 2 "0"

# Calculate weights based on density
$i := 0
while $i < 3
  getarrayelement $densities $i $density
  $weight := 0
  
  # Convert to number for calculation
  add $density 0 $densityNum
  
  # Bad density adds weight (avoid) - density between 1-99 is bad
  if $densityNum > 0
    if $densityNum < 100
      add $weight 100 $weight
      add $weight $densityNum $weight
    end
  end
  
  # Add small randomness (simplified)
  add $weight 5 $weight
  
  setarrayelement $weights $i $weight
  
  getarrayelement $sectors $i $sector
  echo "Sector " $sector " density " $density " weight " $weight
  
  add $i 1 $i
end

echo "Weight calculation completed"
`
	
	result := tester.ExecuteScript(script)
	tester.AssertNoError(result)
	
	// Verify weighting logic
	tester.AssertOutputContains(result, "Calculating sector weights...")
	tester.AssertOutputContains(result, "Sector 1000 density 100")
	tester.AssertOutputContains(result, "Sector 1002 density 0")
	tester.AssertOutputContains(result, "Weight calculation completed")
	
	// Sector 1001 should have higher weight due to bad density (50)
	tester.AssertOutputContains(result, "Sector 1001 density 50 weight 155")
}