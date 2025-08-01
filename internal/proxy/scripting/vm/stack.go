package vm

import (
	"fmt"
	"twist/internal/proxy/scripting/types"
)

// StackFrame represents a frame on the call stack
type StackFrame struct {
	Label      string
	Position   int
	Variables  map[string]*types.Value
	ReturnAddr int
}

// NewStackFrame creates a new stack frame
func NewStackFrame(label string, position int, returnAddr int) *StackFrame {
	return &StackFrame{
		Label:      label,
		Position:   position,
		Variables:  make(map[string]*types.Value),
		ReturnAddr: returnAddr,
	}
}

// CallStack manages the execution call stack for gosub/return operations
type CallStack struct {
	frames []*StackFrame
}

// NewCallStack creates a new call stack
func NewCallStack() *CallStack {
	return &CallStack{
		frames: make([]*StackFrame, 0),
	}
}

// Push adds a new frame to the call stack
func (cs *CallStack) Push(frame *StackFrame) {
	cs.frames = append(cs.frames, frame)
}

// Pop removes and returns the top frame from the call stack
func (cs *CallStack) Pop() (*StackFrame, error) {
	if len(cs.frames) == 0 {
		return nil, fmt.Errorf("call stack is empty")
	}
	
	frame := cs.frames[len(cs.frames)-1]
	cs.frames = cs.frames[:len(cs.frames)-1]
	return frame, nil
}

// Peek returns the top frame without removing it
func (cs *CallStack) Peek() (*StackFrame, error) {
	if len(cs.frames) == 0 {
		return nil, fmt.Errorf("call stack is empty")
	}
	
	return cs.frames[len(cs.frames)-1], nil
}

// IsEmpty returns true if the call stack is empty
func (cs *CallStack) IsEmpty() bool {
	return len(cs.frames) == 0
}

// Size returns the number of frames in the call stack
func (cs *CallStack) Size() int {
	return len(cs.frames)
}

// Clear removes all frames from the call stack
func (cs *CallStack) Clear() {
	cs.frames = cs.frames[:0]
}

// GetFrames returns a copy of all frames (for debugging)
func (cs *CallStack) GetFrames() []*StackFrame {
	frames := make([]*StackFrame, len(cs.frames))
	copy(frames, cs.frames)
	return frames
}