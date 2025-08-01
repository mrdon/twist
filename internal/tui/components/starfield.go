package components

import (
	"math/rand"
	"strings"
	"time"
	"twist/internal/theme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Star represents a single star in 3D space
type Star struct {
	X, Y, Z float64
}

// StarfieldComponent creates an animated starfield effect
type StarfieldComponent struct {
	*tview.Box
	stars       []Star
	numStars    int
	maxDepth    float64
	speed       float64
	width       int
	height      int
	running     bool
	stopChan    chan bool
	updateChan  chan bool
	app         *tview.Application
}

// NewStarfieldComponent creates a new starfield animation
func NewStarfieldComponent(numStars int, app *tview.Application) *StarfieldComponent {
	sf := &StarfieldComponent{
		Box:        tview.NewBox(),
		numStars:   numStars,
		maxDepth:   12.0,
		speed:      0.05,
		stars:      make([]Star, numStars),
		running:    false,
		stopChan:   make(chan bool, 1),
		updateChan: make(chan bool, 1),
		app:        app,
	}
	
	// Use terminal background color from theme
	terminalColors := theme.Current().TerminalColors()
	sf.Box.SetBackgroundColor(terminalColors.Background)
	
	// Match TerminalView's border padding: 1 row top/bottom, 1 column left/right
	sf.Box.SetBorder(false)
	sf.Box.SetBorderPadding(1, 1, 1, 1)
	
	sf.initStars()
	
	return sf
}

// initStars initializes the starfield with random star positions
func (sf *StarfieldComponent) initStars() {
	for i := range sf.stars {
		sf.stars[i] = Star{
			X: (rand.Float64() - 0.5) * 10.0, // -5 to +5
			Y: (rand.Float64() - 0.5) * 10.0, // -5 to +5
			Z: rand.Float64() * sf.maxDepth + 1.0, // 1 to maxDepth
		}
	}
}

// Start begins the starfield animation
func (sf *StarfieldComponent) Start() {
	if sf.running {
		return
	}
	
	sf.running = true
	go sf.animationLoop()
}

// Stop ends the starfield animation
func (sf *StarfieldComponent) Stop() {
	if !sf.running {
		return
	}
	
	sf.running = false
	select {
	case sf.stopChan <- true:
	default:
	}
}

// animationLoop runs the animation in a separate goroutine
func (sf *StarfieldComponent) animationLoop() {
	ticker := time.NewTicker(50 * time.Millisecond) // ~20 FPS
	defer ticker.Stop()
	
	for sf.running {
		select {
		case <-sf.stopChan:
			return
		case <-ticker.C:
			sf.moveStars()
			// Trigger a redraw using the app's QueueUpdateDraw
			if sf.app != nil {
				sf.app.QueueUpdateDraw(func() {
					// The draw will be handled by the Draw method
				})
			}
			// Also signal via channel for compatibility
			select {
			case sf.updateChan <- true:
			default:
			}
		}
	}
}

// moveStars updates star positions for the next frame
func (sf *StarfieldComponent) moveStars() {
	for i := range sf.stars {
		// Move star closer (decrease Z)
		sf.stars[i].Z -= sf.speed
		
		// Reset star if it goes past the screen
		if sf.stars[i].Z <= 0 {
			sf.stars[i].X = (rand.Float64() - 0.5) * 10.0
			sf.stars[i].Y = (rand.Float64() - 0.5) * 10.0
			sf.stars[i].Z = sf.maxDepth
		}
	}
}

// Draw renders the starfield
func (sf *StarfieldComponent) Draw(screen tcell.Screen) {
	sf.Box.DrawForSubclass(screen, sf)
	
	// Get the drawable area
	x, y, width, height := sf.GetInnerRect()
	sf.width = width
	sf.height = height
	
	if width <= 0 || height <= 0 {
		return
	}
	
	// Calculate center point
	centerX := float64(width) / 2.0
	centerY := float64(height) / 2.0
	
	// Draw each star
	for _, star := range sf.stars {
		// Project 3D coordinates to 2D screen coordinates
		screenX := int(centerX + (star.X/star.Z)*centerX)
		screenY := int(centerY + (star.Y/star.Z)*centerY)
		
		// Check if star is within screen bounds
		if screenX >= 0 && screenX < width && screenY >= 0 && screenY < height {
			// Calculate star brightness based on distance (closer = brighter)
			brightness := 1.0 - (star.Z / sf.maxDepth)
			
			// Get terminal colors from theme
			termColors := theme.Current().TerminalColors()
			
			// Choose character and color based on distance using absolute hex colors
			var char rune
			var style tcell.Style
			
			if brightness > 0.8 {
				char = '*'
				// Brightest stars - use white
				brightColor := tcell.NewHexColor(0xFFFFFF)
				style = tcell.StyleDefault.Foreground(brightColor).Background(termColors.Background)
			} else if brightness > 0.6 {
				char = '+'
				// Bright stars - use terminal foreground (light gray)
				style = tcell.StyleDefault.Foreground(termColors.Foreground).Background(termColors.Background)
			} else if brightness > 0.4 {
				char = '.'
				// Medium stars - use medium gray
				mediumColor := tcell.NewHexColor(0x808080)
				style = tcell.StyleDefault.Foreground(mediumColor).Background(termColors.Background)
			} else {
				char = '.'
				// Dim stars - use dark gray
				dimColor := tcell.NewHexColor(0x404040)
				style = tcell.StyleDefault.Foreground(dimColor).Background(termColors.Background)
			}
			
			// Draw the star
			screen.SetContent(x+screenX, y+screenY, char, nil, style)
		}
	}
}

// GetUpdateChannel returns the channel that signals when the component needs redrawing
func (sf *StarfieldComponent) GetUpdateChannel() <-chan bool {
	return sf.updateChan
}

// SetSpeed sets the animation speed
func (sf *StarfieldComponent) SetSpeed(speed float64) {
	sf.speed = speed
}

// StarfieldIntro creates a starfield intro screen that transitions to the main app
type StarfieldIntro struct {
	*tview.Pages
	starfield    *StarfieldComponent
	introDone    chan bool
	app          *tview.Application
	titleText    *tview.TextView
	showingTitle bool
}

// NewStarfieldIntro creates a new starfield intro screen
func NewStarfieldIntro(app *tview.Application) *StarfieldIntro {
	si := &StarfieldIntro{
		Pages:     tview.NewPages(),
		starfield: NewStarfieldComponent(200, app),
		introDone: make(chan bool, 1),
		app:       app,
	}
	
	// Create title text
	si.titleText = tview.NewTextView()
	si.titleText.SetText("")
	si.titleText.SetTextAlign(tview.AlignCenter)
	si.titleText.SetTextColor(tcell.ColorWhite)
	si.titleText.SetBackgroundColor(tcell.ColorBlack)
	si.titleText.SetBorder(false)
	
	// Create a flex container to center the title
	titleContainer := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(si.titleText, 3, 0, false).
		AddItem(nil, 0, 1, false)
	titleContainer.SetBackgroundColor(tcell.ColorBlack)
	
	// Add starfield and title as separate pages
	si.Pages.AddPage("starfield", si.starfield, true, true)
	si.Pages.AddPage("title", titleContainer, true, false)
	
	return si
}

// Start begins the intro sequence
func (si *StarfieldIntro) Start() {
	si.starfield.Start()
	
	// Start the intro sequence
	go si.introSequence()
	
	// Handle starfield updates
	go func() {
		for {
			select {
			case <-si.starfield.GetUpdateChannel():
				si.app.QueueUpdateDraw(func() {
					// Redraw handled automatically
				})
			case <-si.introDone:
				return
			}
		}
	}()
}

// introSequence manages the intro timing and effects
func (si *StarfieldIntro) introSequence() {
	// Phase 1: Show starfield for 2 seconds
	time.Sleep(2 * time.Second)
	
	if !si.showingTitle {
		// Phase 2: Show title over starfield
		si.showingTitle = true
		si.app.QueueUpdateDraw(func() {
			si.titleText.SetText(si.buildTitleText())
			si.Pages.ShowPage("title")
		})
		
		// Phase 3: Keep title visible for 3 seconds
		time.Sleep(3 * time.Second)
	}
	
	// Phase 4: Signal that intro is complete
	select {
	case si.introDone <- true:
	default:
	}
}

// buildTitleText creates the ASCII art title
func (si *StarfieldIntro) buildTitleText() string {
	title := []string{
		"████████╗██╗    ██╗██╗███████╗████████╗",
		"╚══██╔══╝██║    ██║██║██╔════╝╚══██╔══╝",
		"   ██║   ██║ █╗ ██║██║███████╗   ██║   ",
		"   ██║   ██║███╗██║██║╚════██║   ██║   ", 
		"   ██║   ╚███╔███╔╝██║███████║   ██║   ",
		"   ╚═╝    ╚══╝╚══╝ ╚═╝╚══════╝   ╚═╝   ",
		"",
		"    Trade Wars Terminal Interface",
	}
	
	return strings.Join(title, "\n")
}

// GetDoneChannel returns the channel that signals when the intro is complete
func (si *StarfieldIntro) GetDoneChannel() <-chan bool {
	return si.introDone
}

// Stop stops the intro
func (si *StarfieldIntro) Stop() {
	si.starfield.Stop()
	select {
	case si.introDone <- true:
	default:
	}
}