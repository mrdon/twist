package components

import (
	"fmt"
	"strings"
	"twist/internal/database"
	"twist/internal/theme"
	
	"github.com/rivo/tview"
)

// PanelComponent manages the side panel components
type PanelComponent struct {
	leftView    *tview.TextView
	leftWrapper *tview.Flex
	rightView   *tview.TextView
	rightWrapper *tview.Flex
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

// UpdateTraderInfo updates the trader information panel
func (pc *PanelComponent) UpdateTraderInfo(trader *database.TTrader) {
	if trader == nil {
		pc.leftView.SetText("No trader data")
		return
	}
	
	var info strings.Builder
	info.WriteString(fmt.Sprintf("[yellow]Trader Info[-]\n"))
	info.WriteString(fmt.Sprintf("Name: %s\n", trader.Name))
	info.WriteString(fmt.Sprintf("Ship: %s\n", trader.ShipType))
	if trader.Figs > 0 {
		info.WriteString(fmt.Sprintf("Fighters: %d\n", trader.Figs))
	}
	
	pc.leftView.SetText(info.String())
}

// UpdateSectorInfo updates the sector information panel
func (pc *PanelComponent) UpdateSectorInfo(sector *database.TSector) {
	if sector == nil {
		pc.rightView.SetText("No sector data")
		return
	}
	
	var info strings.Builder
	info.WriteString(fmt.Sprintf("[cyan]Sector Info[-]\n"))
	info.WriteString(fmt.Sprintf("Nav Hazard: %d\n", sector.NavHaz))
	
	if sector.Figs.Quantity > 0 {
		info.WriteString(fmt.Sprintf("Fighters: %d\n", sector.Figs.Quantity))
	}
	
	if sector.MinesArmid.Quantity > 0 {
		info.WriteString(fmt.Sprintf("Armid Mines: %d\n", sector.MinesArmid.Quantity))
	}
	
	if sector.MinesLimpet.Quantity > 0 {
		info.WriteString(fmt.Sprintf("Limpet Mines: %d\n", sector.MinesLimpet.Quantity))
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