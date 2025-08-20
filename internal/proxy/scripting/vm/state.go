package vm

// ExecutionState represents the current state of the virtual machine
type ExecutionState int

const (
	StateRunning ExecutionState = iota
	StatePaused
	StateHalted
	StateWaiting
	StateError
)

// VMState holds the execution state of the virtual machine
type VMState struct {
	State      ExecutionState
	Position   int
	Error      string
	Waiting    bool
	WaitText   string
	JumpTarget string
}

// NewVMState creates a new VM state
func NewVMState() *VMState {
	return &VMState{
		State: StateHalted,
	}
}

// IsRunning returns true if the VM is currently running
func (s *VMState) IsRunning() bool {
	return s.State == StateRunning
}

// IsPaused returns true if the VM is currently paused
func (s *VMState) IsPaused() bool {
	return s.State == StatePaused
}

// IsHalted returns true if the VM is currently halted
func (s *VMState) IsHalted() bool {
	return s.State == StateHalted
}

// IsWaiting returns true if the VM is waiting for input
func (s *VMState) IsWaiting() bool {
	return s.State == StateWaiting || s.Waiting
}

// HasError returns true if the VM is in an error state
func (s *VMState) HasError() bool {
	return s.State == StateError
}

// SetRunning sets the VM to running state
func (s *VMState) SetRunning() {
	s.State = StateRunning
	s.Error = ""
}

// SetPaused sets the VM to paused state
func (s *VMState) SetPaused() {
	s.State = StatePaused
}

// SetHalted sets the VM to halted state
func (s *VMState) SetHalted() {
	s.State = StateHalted
}

// SetWaiting sets the VM to waiting state with optional wait text
func (s *VMState) SetWaiting(waitText string) {
	s.State = StateWaiting
	s.Waiting = true
	s.WaitText = waitText
}

// SetError sets the VM to error state with error message
func (s *VMState) SetError(errorMsg string) {
	s.State = StateError
	s.Error = errorMsg
}

// ClearWait clears the waiting state
func (s *VMState) ClearWait() {
	s.Waiting = false
	s.WaitText = ""
	if s.State == StateWaiting {
		s.State = StateRunning
	}
}

// SetJumpTarget sets the target for a jump operation
func (s *VMState) SetJumpTarget(target string) {
	s.JumpTarget = target
}

// ClearJumpTarget clears the jump target
func (s *VMState) ClearJumpTarget() {
	s.JumpTarget = ""
}

// HasJumpTarget returns true if there's a pending jump
func (s *VMState) HasJumpTarget() bool {
	return s.JumpTarget != ""
}
