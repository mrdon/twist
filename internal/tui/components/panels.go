package components

import (
	"fmt"
	"strings"
	"twist/internal/api"
	"twist/internal/theme"
	
	"github.com/rivo/tview"
)

// PanelComponent manages the side panel components
type PanelComponent struct {
	leftView     *tview.TextView
	leftWrapper  *tview.Flex
	rightWrapper *tview.Flex
	sectorMap    *SectorMapComponent  // New sector map component
	proxyAPI     api.ProxyAPI  // API access for game data
}

// NewPanelComponent creates new panel components
func NewPanelComponent() *PanelComponent {
	// Left panel for trader info using theme
	leftPanel := theme.NewPanelView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	leftPanel.SetBorder(true).SetTitle("Trader Info")
	leftPanel.SetText("[yellow]Player Info[-]\n\n[cyan]Connect to see trader info[-]")
	
	leftWrapper := theme.NewFlex().SetDirection(tview.FlexRow).
		AddItem(leftPanel, 0, 1, false)
	
	// Create sector map component for right panel
	sectorMap := NewSectorMapComponent()
	
	// Right panel is just the sector map
	rightWrapper := theme.NewFlex().SetDirection(tview.FlexRow).
		AddItem(sectorMap.GetView(), 0, 1, false)
	
	return &PanelComponent{
		leftView:     leftPanel,
		leftWrapper:  leftWrapper,
		rightWrapper: rightWrapper,
		sectorMap:    sectorMap,
	}
}

// GetLeftWrapper returns the left panel wrapper
func (pc *PanelComponent) GetLeftWrapper() *tview.Flex {
	return pc.leftWrapper
}

// GetRightWrapper returns the right panel wrapper
func (pc *PanelComponent) GetRightWrapper() *tview.Flex {
	return pc.rightWrapper
}

// SetProxyAPI sets the API reference for accessing game data
func (pc *PanelComponent) SetProxyAPI(proxyAPI api.ProxyAPI) {
	pc.proxyAPI = proxyAPI
	if pc.sectorMap != nil {
		pc.sectorMap.SetProxyAPI(proxyAPI)
	}
	
	// Show test trader info when API is connected
	if proxyAPI != nil {
		pc.showTestTraderInfo()
	}
}

// showTestTraderInfo displays test trader information
func (pc *PanelComponent) showTestTraderInfo() {
	var info strings.Builder
	info.WriteString("[yellow]Player Info[-]\n\n")
	info.WriteString("Name: TestPlayer\n")
	info.WriteString("Current Sector: 123\n")
	info.WriteString("Ship: Imperial StarShip\n")
	info.WriteString("Credits: 50,000\n")
	info.WriteString("Fighters: 100\n")
	info.WriteString("Holds: 50/50\n\n")
	info.WriteString("[cyan]Cargo[-]\n")
	info.WriteString("Fuel Ore: 10\n")
	info.WriteString("Organics: 15\n")
	info.WriteString("Equipment: 25\n")
	
	pc.leftView.SetText(info.String())
}

// UpdateTraderInfo updates the trader information panel using API PlayerInfo
func (pc *PanelComponent) UpdateTraderInfo(playerInfo api.PlayerInfo) {
	var info strings.Builder
	info.WriteString(fmt.Sprintf("[yellow]Player Info[-]\n"))
	
	if playerInfo.Name != "" {
		info.WriteString(fmt.Sprintf("Name: %s\n", playerInfo.Name))
	}
	
	info.WriteString(fmt.Sprintf("Current Sector: %d\n", playerInfo.CurrentSector))
	
	pc.leftView.SetText(info.String())
}

// UpdateSectorInfo updates the sector map with current sector info
func (pc *PanelComponent) UpdateSectorInfo(sector api.SectorInfo) {
	if pc.sectorMap != nil {
		pc.sectorMap.UpdateCurrentSector(sector.Number)
	}
}

// SetTraderInfoText sets custom text in the trader info panel
func (pc *PanelComponent) SetTraderInfoText(text string) {
	pc.leftView.SetText(text)
}

// SetPlaceholderPlayerText sets placeholder text for testing Phase 4.1
func (pc *PanelComponent) SetPlaceholderPlayerText() {
	pc.leftView.SetText("[yellow]Player Info[-]\nAPI data not yet available")
}