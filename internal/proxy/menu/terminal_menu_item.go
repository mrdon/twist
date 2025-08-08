package menu

import (
	"twist/internal/debug"
)

type TerminalMenuHandler func(*TerminalMenuItem, []string) error

type TerminalMenuItem struct {
	Name        string
	Description string
	Hotkey      rune
	Parent      *TerminalMenuItem
	Children    []*TerminalMenuItem
	Handler     TerminalMenuHandler
	Parameters  []string
	Reference   string
	Prompt      string
	CloseMenu   bool
	ScriptOwner string // Script ID that owns this menu
}

func NewTerminalMenuItem(name, description string, hotkey rune) *TerminalMenuItem {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in NewTerminalMenuItem: %v", r)
		}
	}()

	return &TerminalMenuItem{
		Name:        name,
		Description: description,
		Hotkey:      hotkey,
		Children:    make([]*TerminalMenuItem, 0),
		Parameters:  make([]string, 0),
	}
}

func (item *TerminalMenuItem) AddChild(child *TerminalMenuItem) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in AddChild: %v", r)
		}
	}()

	if child == nil {
		debug.Log("Warning: Attempted to add nil child to menu item %s", item.Name)
		return
	}

	child.Parent = item
	item.Children = append(item.Children, child)
}

func (item *TerminalMenuItem) RemoveChild(child *TerminalMenuItem) bool {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in RemoveChild: %v", r)
		}
	}()

	for i, c := range item.Children {
		if c == child {
			item.Children = append(item.Children[:i], item.Children[i+1:]...)
			child.Parent = nil
			return true
		}
	}
	return false
}

func (item *TerminalMenuItem) FindChildByHotkey(hotkey rune) *TerminalMenuItem {
	for _, child := range item.Children {
		if child.Hotkey == hotkey {
			return child
		}
	}
	return nil
}

func (item *TerminalMenuItem) FindChildByName(name string) *TerminalMenuItem {
	for _, child := range item.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

func (item *TerminalMenuItem) GetPath() string {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in GetPath: %v", r)
		}
	}()

	if item.Parent == nil {
		return item.Name
	}
	return item.Parent.GetPath() + " > " + item.Name
}

func (item *TerminalMenuItem) IsRoot() bool {
	return item.Parent == nil
}

func (item *TerminalMenuItem) HasChildren() bool {
	return len(item.Children) > 0
}

func (item *TerminalMenuItem) Execute(params []string) error {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in Execute: %v", r)
		}
	}()

	if item.Handler == nil {
		debug.Log("No handler defined for menu item: %s", item.Name)
		return nil
	}

	return item.Handler(item, params)
}

func (item *TerminalMenuItem) Clone() *TerminalMenuItem {
	defer func() {
		if r := recover(); r != nil {
			debug.Log("PANIC in Clone: %v", r)
		}
	}()

	clone := &TerminalMenuItem{
		Name:        item.Name,
		Description: item.Description,
		Hotkey:      item.Hotkey,
		Handler:     item.Handler,
		Reference:   item.Reference,
		Prompt:      item.Prompt,
		CloseMenu:   item.CloseMenu,
		ScriptOwner: item.ScriptOwner,
		Parameters:  make([]string, len(item.Parameters)),
		Children:    make([]*TerminalMenuItem, 0),
	}

	copy(clone.Parameters, item.Parameters)

	for _, child := range item.Children {
		clone.AddChild(child.Clone())
	}

	return clone
}