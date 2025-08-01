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
	rightView    *tview.TextView
	rightWrapper *tview.Flex
	proxyAPI     api.ProxyAPI  // API access for game data
}

// NewPanelComponent creates new panel components
func NewPanelComponent() *PanelComponent {
	// Left panel for trader info using theme
	leftPanel := theme.NewPanelView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	leftPanel.SetBorder(true).SetTitle("Trader Info")
	
	leftWrapper := theme.NewFlex().SetDirection(tview.FlexRow).
		AddItem(leftPanel, 0, 1, false)
	
	// Right panel for sector info using theme
	rightPanel := theme.NewPanelView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	rightPanel.SetBorder(true).SetTitle("Sector Info")
	
	rightWrapper := theme.NewFlex().SetDirection(tview.FlexRow).
		AddItem(rightPanel, 0, 1, false)
	
	return &PanelComponent{
		leftView:     leftPanel,
		leftWrapper:  leftWrapper,
		rightView:    rightPanel,
		rightWrapper: rightWrapper,
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

// UpdateSectorInfo updates the sector information panel using API SectorInfo
func (pc *PanelComponent) UpdateSectorInfo(sector api.SectorInfo) {
	var info strings.Builder
	info.WriteString(fmt.Sprintf("[cyan]Sector %d Info[-]\n", sector.Number))
	info.WriteString(fmt.Sprintf("Nav Hazard: %d\n", sector.NavHaz))
	
	if sector.HasTraders > 0 {
		info.WriteString(fmt.Sprintf("Traders: %d\n", sector.HasTraders))
	}
	
	if sector.Constellation != "" {
		info.WriteString(fmt.Sprintf("Constellation: %s\n", sector.Constellation))
	}
	
	if sector.Beacon != "" {
		info.WriteString(fmt.Sprintf("Beacon: %s\n", sector.Beacon))
	}
	
	pc.rightView.SetText(info.String())
}

// SetTraderInfoText sets custom text in the trader info panel
func (pc *PanelComponent) SetTraderInfoText(text string) {
	pc.leftView.SetText(text)
}

// SetSectorInfoText sets custom text in the sector info panel  
func (pc *PanelComponent) SetSectorInfoText(text string) {
	pc.rightView.SetText(text)
}

// SetPlaceholderPlayerText sets placeholder text for testing Phase 4.1
func (pc *PanelComponent) SetPlaceholderPlayerText() {
	pc.leftView.SetText("[yellow]Player Info[-]\nAPI data not yet available")
}

// SetPlaceholderSectorText sets placeholder text for testing Phase 4.1
func (pc *PanelComponent) SetPlaceholderSectorText() {
	pc.rightView.SetText("[cyan]Sector Info[-]\nAPI data not yet available")
}