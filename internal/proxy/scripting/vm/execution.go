package vm

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"twist/internal/proxy/scripting/parser"
	"twist/internal/proxy/scripting/types"
)

// ExecutionEngine handles the execution of parsed script commands
type ExecutionEngine struct {
	vm  *VirtualMachine
	ast *parser.ASTNode
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(vm *VirtualMachine) *ExecutionEngine {
	return &ExecutionEngine{
		vm: vm,
	}
}

// LoadAST loads an AST for execution
func (ee *ExecutionEngine) LoadAST(ast *parser.ASTNode) {
	ee.ast = ast
}

// ExecuteStep executes a single step of the script
func (ee *ExecutionEngine) ExecuteStep() (retErr error) {
	// Add panic recovery for bounds checking and other runtime errors
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("%v", r)
			ee.vm.state.SetError(retErr.Error())
		}
	}()

	if ee.ast == nil || ee.vm.state.Position >= len(ee.ast.Children) {
		ee.vm.state.SetHalted()
		return nil
	}

	node := ee.ast.Children[ee.vm.state.Position]
	err := ee.executeNode(node)

	// Handle script pause (like TWX caPause) - don't advance position for input pauses
	if err != nil && errors.Is(err, types.ErrScriptPaused) {
		ee.vm.state.SetPaused()
		// Don't advance position - we need to re-execute this command when resumed
		return nil // Don't treat pause as an error
	}

	if err != nil {
		ee.vm.state.SetError(err.Error())
		return err
	}

	// Handle jump targets
	if ee.vm.state.HasJumpTarget() {
		newPos := ee.findLabel(ee.vm.state.JumpTarget)
		if newPos == -1 {
			return fmt.Errorf("label not found: %s", ee.vm.state.JumpTarget)
		}
		ee.vm.state.Position = newPos
		ee.vm.state.ClearJumpTarget()
	} else {
		ee.vm.state.Position++
	}

	return nil
}

// executeNode executes a single AST node
func (ee *ExecutionEngine) executeNode(node *parser.ASTNode) error {
	switch node.Type {
	case parser.NodeCommand:
		return ee.executeCommand(node)
	case parser.NodeIf:
		return ee.executeIf(node)
	case parser.NodeWhile:
		return ee.executeWhile(node)
	case parser.NodeAssignment:
		return ee.executeAssignment(node)
	case parser.NodeCompoundAssignment:
		return ee.executeCompoundAssignment(node)
	case parser.NodeIncrementDecrement:
		return ee.executeIncrementDecrement(node)
	case parser.NodeInclude:
		// INCLUDE nodes should be preprocessed out before execution
		return fmt.Errorf("INCLUDE node found during execution - should have been preprocessed")
	case parser.NodeLabel:
		// Labels are processed during execution flow, no action needed
		return nil
	default:
		return fmt.Errorf("unknown node type: %v", node.Type)
	}
}

// executeCommand executes a command node
func (ee *ExecutionEngine) executeCommand(node *parser.ASTNode) error {
	// The command name is in the node's Value, not in children
	cmdName := node.Value
	if cmdName == "" {
		return fmt.Errorf("command node has no command name")
	}

	cmdDef, exists := ee.vm.commands[strings.ToUpper(cmdName)]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	// Parse parameters - all children are parameters
	params, err := ee.parseCommandParams(node.Children)
	if err != nil {
		return fmt.Errorf("error parsing parameters for %s: %v", cmdName, err)
	}

	// Validate parameter count
	if len(params) < cmdDef.MinParams || (cmdDef.MaxParams != -1 && len(params) > cmdDef.MaxParams) {
		maxParamStr := fmt.Sprintf("%d", cmdDef.MaxParams)
		if cmdDef.MaxParams == -1 {
			maxParamStr = "unlimited"
		}
		return fmt.Errorf("command %s expects %d-%s parameters, got %d",
			cmdName, cmdDef.MinParams, maxParamStr, len(params))
	}

	// Execute the command
	err = cmdDef.Handler(ee.vm, params)

	return err
}

// executeGoto executes a goto statement
func (ee *ExecutionEngine) executeGoto(node *parser.ASTNode) error {
	if len(node.Children) == 0 {
		return fmt.Errorf("goto statement missing label")
	}

	label := node.Children[0].Value
	ee.vm.state.SetJumpTarget(label)
	return nil
}

// executeGosub executes a gosub statement
func (ee *ExecutionEngine) executeGosub(node *parser.ASTNode) error {
	if len(node.Children) == 0 {
		return fmt.Errorf("gosub statement missing label")
	}

	label := node.Children[0].Value

	// Push current position onto call stack
	returnAddr := ee.vm.state.Position + 1
	frame := NewStackFrame(label, ee.vm.state.Position, returnAddr)
	ee.vm.callStack.Push(frame)

	// Jump to the subroutine
	ee.vm.state.SetJumpTarget(label)
	return nil
}

// executeReturn executes a return statement
func (ee *ExecutionEngine) executeReturn(node *parser.ASTNode) error {
	frame, err := ee.vm.callStack.Pop()
	if err != nil {
		return fmt.Errorf("return without gosub")
	}

	// Return to the position after the gosub
	ee.vm.state.Position = frame.ReturnAddr
	return nil
}

// executeIf executes an if statement
func (ee *ExecutionEngine) executeIf(node *parser.ASTNode) error {
	if len(node.Children) < 1 {
		return fmt.Errorf("if statement requires condition")
	}

	// Handle different IF node types
	if node.Value == "else" {
		// This is an else clause - execute all children
		for _, child := range node.Children {
			err := ee.executeNode(child)
			if err != nil {
				return err
			}
		}
		return nil
	}

	if node.Value == "elseif" {
		// This is an elseif clause - first child is condition, rest are statements
		if len(node.Children) < 2 {
			return fmt.Errorf("elseif statement requires condition and action")
		}

		condition, err := ee.evaluateExpression(node.Children[0])
		if err != nil {
			return fmt.Errorf("error evaluating elseif condition: %v", err)
		}

		isTrue := false
		if condition.Type == types.NumberType {
			isTrue = condition.Number != 0
		} else {
			isTrue = condition.String != ""
		}

		if isTrue {
			// Execute the statements
			for i := 1; i < len(node.Children); i++ {
				err := ee.executeNode(node.Children[i])
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	// This is a main IF statement
	// Evaluate condition
	condition, err := ee.evaluateExpression(node.Children[0])
	if err != nil {
		return fmt.Errorf("error evaluating if condition: %v", err)
	}

	// Check if condition is true
	isTrue := false
	if condition.Type == types.NumberType {
		isTrue = condition.Number != 0
	} else {
		isTrue = condition.String != ""
	}

	if isTrue {
		// Execute the then statements - skip condition (index 0) and any else clauses
		for i := 1; i < len(node.Children); i++ {
			child := node.Children[i]
			// Skip else/elseif clauses
			if child.Type == parser.NodeIf && (child.Value == "else" || child.Value == "elseif") {
				break
			}
			err := ee.executeNode(child)
			if err != nil {
				return err
			}
		}
	} else {
		// Look for else/elseif clauses
		executed := false
		for i := 1; i < len(node.Children) && !executed; i++ {
			child := node.Children[i]
			if child.Type == parser.NodeIf {
				if child.Value == "elseif" {
					// Check the elseif condition
					if len(child.Children) < 1 {
						return fmt.Errorf("elseif statement requires condition")
					}

					elseifCondition, err := ee.evaluateExpression(child.Children[0])
					if err != nil {
						return fmt.Errorf("error evaluating elseif condition: %v", err)
					}

					elseifIsTrue := false
					if elseifCondition.Type == types.NumberType {
						elseifIsTrue = elseifCondition.Number != 0
					} else {
						elseifIsTrue = elseifCondition.String != ""
					}

					if elseifIsTrue {
						// Execute the elseif statements
						for j := 1; j < len(child.Children); j++ {
							err := ee.executeNode(child.Children[j])
							if err != nil {
								return err
							}
						}
						executed = true
					}
				} else if child.Value == "else" {
					// Execute the else clause
					for _, elseChild := range child.Children {
						err := ee.executeNode(elseChild)
						if err != nil {
							return err
						}
					}
					executed = true
				}
			}
		}
	}

	return nil
}

// executeWhile executes a while loop
func (ee *ExecutionEngine) executeWhile(node *parser.ASTNode) error {
	if len(node.Children) < 2 {
		return fmt.Errorf("while statement requires condition and body")
	}

	for {
		// Evaluate condition
		condition, err := ee.evaluateExpression(node.Children[0])
		if err != nil {
			return fmt.Errorf("error evaluating while condition: %v", err)
		}

		// Check if condition is true
		isTrue := false
		if condition.Type == types.NumberType {
			isTrue = condition.Number != 0
		} else {
			isTrue = condition.String != ""
		}

		if !isTrue {
			break
		}

		// Execute all statements in the body (skip condition at index 0)
		for i := 1; i < len(node.Children); i++ {
			// Save the current position in case we need to handle jumps
			savedPos := ee.vm.state.Position

			err = ee.executeNode(node.Children[i])
			if err != nil {
				return err
			}

			// Handle jump targets (like GOSUB) using the same logic as ExecuteStep
			if ee.vm.state.HasJumpTarget() {
				newPos := ee.findLabel(ee.vm.state.JumpTarget)
				if newPos == -1 {
					return fmt.Errorf("label not found: %s", ee.vm.state.JumpTarget)
				}

				ee.vm.state.Position = newPos
				ee.vm.state.ClearJumpTarget()

				// Execute from the jump target until we return or reach end
				initialCallStackSize := ee.vm.callStack.Size()

				for ee.vm.state.Position < len(ee.ast.Children) && ee.vm.state.IsRunning() {
					currentNode := ee.ast.Children[ee.vm.state.Position]
					err = ee.executeNode(currentNode)
					if err != nil {
						return err
					}

					// Check if a RETURN command was executed by seeing if call stack shrank
					if ee.vm.callStack.Size() < initialCallStackSize {
						break
					}

					// Handle any new jump targets that might have been set
					if ee.vm.state.HasJumpTarget() {
						break
					}

					ee.vm.state.Position++
				}

				// Restore the saved position for the while loop
				ee.vm.state.Position = savedPos
			}
		}
	}

	return nil
}

// executeAssignment executes a variable assignment
func (ee *ExecutionEngine) executeAssignment(node *parser.ASTNode) error {
	if len(node.Children) < 2 {
		return fmt.Errorf("assignment requires variable name and value")
	}

	varName := node.Children[0].Value
	value, err := ee.evaluateExpression(node.Children[1])
	if err != nil {
		return fmt.Errorf("error evaluating assignment value: %v", err)
	}

	ee.vm.variables.Set(varName, value)
	return nil
}

// executeCompoundAssignment executes compound assignment operations (+=, -=, *=, /=, &=)
func (ee *ExecutionEngine) executeCompoundAssignment(node *parser.ASTNode) error {
	if len(node.Children) < 2 {
		return fmt.Errorf("compound assignment requires variable name and value")
	}

	// Get variable name - handle both direct variables and array access
	var varName string
	if node.Children[0].Type == parser.NodeVariable {
		varName = node.Children[0].Value
	} else if node.Children[0].Type == parser.NodeArrayAccess {
		// For now, just handle simple array access
		varName = node.Children[0].Children[0].Value
	} else {
		return fmt.Errorf("invalid variable in compound assignment")
	}

	current := ee.vm.variables.Get(varName)

	// Evaluate the right-hand side
	rhsValue, err := ee.evaluateExpression(node.Children[1])
	if err != nil {
		return fmt.Errorf("error evaluating compound assignment value: %v", err)
	}

	var result *types.Value
	switch node.Value {
	case "+=":
		num := current.ToNumber() + rhsValue.ToNumber()
		result = &types.Value{
			Type:   types.NumberType,
			Number: num,
			String: fmt.Sprintf("%g", num),
		}
	case "-=":
		num := current.ToNumber() - rhsValue.ToNumber()
		result = &types.Value{
			Type:   types.NumberType,
			Number: num,
			String: fmt.Sprintf("%g", num),
		}
	case "*=":
		num := current.ToNumber() * rhsValue.ToNumber()
		result = &types.Value{
			Type:   types.NumberType,
			Number: num,
			String: fmt.Sprintf("%g", num),
		}
	case "/=":
		if rhsValue.ToNumber() == 0 {
			return fmt.Errorf("division by zero in compound assignment")
		}
		num := current.ToNumber() / rhsValue.ToNumber()
		result = &types.Value{
			Type:   types.NumberType,
			Number: num,
			String: fmt.Sprintf("%g", num),
		}
	case "&=":
		str := current.String + rhsValue.String
		result = &types.Value{
			Type:   types.StringType,
			String: str,
		}
	default:
		return fmt.Errorf("unsupported compound assignment operator: %s", node.Value)
	}

	ee.vm.variables.Set(varName, result)
	return nil
}

// executeIncrementDecrement executes increment/decrement operations (++, --)
func (ee *ExecutionEngine) executeIncrementDecrement(node *parser.ASTNode) error {
	if len(node.Children) < 1 {
		return fmt.Errorf("increment/decrement requires variable name")
	}

	// Get variable name - handle both direct variables and array access
	var varName string
	if node.Children[0].Type == parser.NodeVariable {
		varName = node.Children[0].Value
	} else if node.Children[0].Type == parser.NodeArrayAccess {
		// For now, just handle simple array access
		varName = node.Children[0].Children[0].Value
	} else {
		return fmt.Errorf("invalid variable in increment/decrement")
	}

	current := ee.vm.variables.Get(varName)

	var result *types.Value
	switch node.Value {
	case "++":
		num := current.ToNumber() + 1
		result = &types.Value{
			Type:   types.NumberType,
			Number: num,
			String: fmt.Sprintf("%g", num),
		}
	case "--":
		num := current.ToNumber() - 1
		result = &types.Value{
			Type:   types.NumberType,
			Number: num,
			String: fmt.Sprintf("%g", num),
		}
	default:
		return fmt.Errorf("unsupported increment/decrement operator: %s", node.Value)
	}

	ee.vm.variables.Set(varName, result)
	return nil
}

// findLabel finds the position of a label in the AST
func (ee *ExecutionEngine) findLabel(label string) int {
	if ee.ast == nil {
		return -1
	}

	// Normalize the target label (remove colon, convert to lowercase)
	targetLabel := strings.ToLower(strings.TrimPrefix(label, ":"))

	for i, node := range ee.ast.Children {
		if node.Type == parser.NodeLabel {
			// Normalize the node label (remove colon, convert to lowercase)
			nodeLabel := strings.ToLower(strings.TrimPrefix(node.Value, ":"))
			if nodeLabel == targetLabel {
				return i
			}
		}
	}
	return -1
}

// parseCommandParams parses command parameters from AST nodes
func (ee *ExecutionEngine) parseCommandParams(nodes []*parser.ASTNode) ([]*types.CommandParam, error) {
	params := make([]*types.CommandParam, len(nodes))

	for i, node := range nodes {
		if node.Type == parser.NodeVariable {
			params[i] = &types.CommandParam{
				Type:    types.ParamVar,
				VarName: node.Value,
			}
		} else if node.Type == parser.NodeArrayAccess {
			// Handle array access as variable parameter
			arrayVarName, err := ee.buildArrayVarName(node)
			if err != nil {
				return nil, fmt.Errorf("error building array variable name: %v", err)
			}
			params[i] = &types.CommandParam{
				Type:    types.ParamVar,
				VarName: arrayVarName,
			}
		} else {
			value, err := ee.evaluateExpression(node)
			if err != nil {
				return nil, err
			}
			params[i] = &types.CommandParam{
				Type:  types.ParamValue,
				Value: value,
			}
		}
	}

	return params, nil
}

// evaluateExpression evaluates an expression node and returns its value
func (ee *ExecutionEngine) evaluateExpression(node *parser.ASTNode) (*types.Value, error) {
	switch node.Type {
	case parser.NodeLiteral:
		// Try to parse as number first
		if num, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return &types.Value{
				Type:   types.NumberType,
				Number: num,
				String: node.Value,
			}, nil
		}
		// Otherwise treat as string
		return &types.Value{
			Type:   types.StringType,
			String: node.Value,
			Number: 0,
		}, nil

	case parser.NodeVariable:
		return ee.vm.variables.Get(node.Value), nil

	case parser.NodeExpression:
		// Check if this is a unary expression (single child) or binary expression (two children)
		if len(node.Children) == 1 {
			return ee.evaluateUnaryExpression(node)
		} else {
			return ee.evaluateBinaryExpression(node)
		}

	case parser.NodeCommand:
		// When a command name appears as a parameter (e.g., in GOTO), treat it as a literal string
		return &types.Value{
			Type:   types.StringType,
			String: node.Value,
		}, nil

	case parser.NodeArrayAccess:
		// Handle array access like $sectors[1] or $data[1][2]
		return ee.evaluateArrayAccess(node)

	default:
		return nil, fmt.Errorf("cannot evaluate node type: %v", node.Type)
	}
}

// buildArrayVarName builds a variable name string from an array access node
func (ee *ExecutionEngine) buildArrayVarName(node *parser.ASTNode) (string, error) {
	if node.Type != parser.NodeArrayAccess {
		return "", fmt.Errorf("node is not an array access")
	}

	if len(node.Children) < 2 {
		return "", fmt.Errorf("array access requires variable and at least one index")
	}

	// First child should be the base variable
	if node.Children[0].Type != parser.NodeVariable {
		return "", fmt.Errorf("array access base must be a variable")
	}

	baseName := node.Children[0].Value

	// Evaluate all index expressions and build the variable name
	var varName strings.Builder
	varName.WriteString(baseName)

	for i := 1; i < len(node.Children); i++ {
		indexValue, err := ee.evaluateExpression(node.Children[i])
		if err != nil {
			return "", fmt.Errorf("error evaluating array index %d: %v", i, err)
		}
		varName.WriteString("[")
		varName.WriteString(indexValue.ToString())
		varName.WriteString("]")
	}

	return varName.String(), nil
}

// evaluateArrayAccess evaluates array access expressions like $sectors[1] or $data[1][2]
func (ee *ExecutionEngine) evaluateArrayAccess(node *parser.ASTNode) (*types.Value, error) {
	if len(node.Children) < 2 {
		return nil, fmt.Errorf("array access requires variable and at least one index")
	}

	// First child should be the base variable
	if node.Children[0].Type != parser.NodeVariable {
		return nil, fmt.Errorf("array access base must be a variable")
	}

	baseName := node.Children[0].Value

	// Get or create VarParam for this variable
	varParam := ee.vm.variables.GetVarParam(baseName)
	if varParam == nil {
		// Auto-vivification: create new VarParam
		varParam = types.NewVarParam(baseName, types.VarParamVariable)
		ee.vm.variables.SetVarParam(baseName, varParam)
	}

	// Evaluate all index expressions and build index path
	indexes := make([]string, len(node.Children)-1)
	for i := 1; i < len(node.Children); i++ {
		indexValue, err := ee.evaluateExpression(node.Children[i])
		if err != nil {
			return nil, fmt.Errorf("error evaluating array index %d: %v", i, err)
		}
		indexes[i-1] = indexValue.ToString()
	}

	// Get the indexed variable
	indexedVar := varParam.GetIndexVar(indexes)

	// Return the value
	return &types.Value{
		Type:   types.StringType,
		String: indexedVar.GetValue(),
	}, nil
}

// evaluateUnaryExpression evaluates a unary operation
func (ee *ExecutionEngine) evaluateUnaryExpression(node *parser.ASTNode) (*types.Value, error) {
	if len(node.Children) != 1 {
		return nil, fmt.Errorf("unary operation requires exactly one operand")
	}

	operand, err := ee.evaluateExpression(node.Children[0])
	if err != nil {
		return nil, err
	}

	operator := node.Value

	switch operator {
	case "-":
		// Unary minus
		if operand.Type == types.NumberType {
			return &types.Value{
				Type:   types.NumberType,
				Number: -operand.Number,
				String: fmt.Sprintf("%g", -operand.Number),
			}, nil
		}
		// Try to convert string to number for unary minus
		if num, err := strconv.ParseFloat(operand.String, 64); err == nil {
			return &types.Value{
				Type:   types.NumberType,
				Number: -num,
				String: fmt.Sprintf("%g", -num),
			}, nil
		}
		return nil, fmt.Errorf("cannot apply unary minus to non-numeric value: %s", operand.String)

	case "+":
		// Unary plus (identity operation)
		if operand.Type == types.NumberType {
			return operand, nil
		}
		// Try to convert string to number for unary plus
		if num, err := strconv.ParseFloat(operand.String, 64); err == nil {
			return &types.Value{
				Type:   types.NumberType,
				Number: num,
				String: fmt.Sprintf("%g", num),
			}, nil
		}
		return nil, fmt.Errorf("cannot apply unary plus to non-numeric value: %s", operand.String)

	case "NOT":
		// Logical NOT
		truthValue := ee.isTruthyValue(operand)
		return &types.Value{
			Type: types.NumberType,
			Number: func() float64 {
				if truthValue {
					return 0
				} else {
					return 1
				}
			}(),
			String: func() string {
				if truthValue {
					return "0"
				} else {
					return "1"
				}
			}(),
		}, nil

	default:
		return nil, fmt.Errorf("unknown unary operator: %s", operator)
	}
}

// isTruthyValue determines if a value is truthy (non-zero for numbers, non-empty for strings)
func (ee *ExecutionEngine) isTruthyValue(value *types.Value) bool {
	switch value.Type {
	case types.NumberType:
		return value.Number != 0
	case types.StringType:
		return value.String != ""
	default:
		return false
	}
}

// evaluateBinaryExpression evaluates a binary operation
func (ee *ExecutionEngine) evaluateBinaryExpression(node *parser.ASTNode) (*types.Value, error) {
	if len(node.Children) < 2 {
		return nil, fmt.Errorf("binary operation requires two operands")
	}

	left, err := ee.evaluateExpression(node.Children[0])
	if err != nil {
		return nil, err
	}

	right, err := ee.evaluateExpression(node.Children[1])
	if err != nil {
		return nil, err
	}

	operator := node.Value

	// For comparison operators, try numeric first (TWX behavior)
	if operator == "<" || operator == ">" || operator == "<=" || operator == ">=" {
		return ee.evaluateNumericOperation(left, right, operator)
	}

	// & operator always does string concatenation in TWX
	if operator == "&" {
		leftStr := left.ToString()
		rightStr := right.ToString()
		return &types.Value{
			Type:   types.StringType,
			String: leftStr + rightStr,
		}, nil
	}

	// Handle string comparison/concatenation
	if left.Type == types.StringType || right.Type == types.StringType {
		return ee.evaluateStringOperation(left, right, operator)
	}

	// Handle numeric operations
	return ee.evaluateNumericOperation(left, right, operator)
}

// evaluateStringOperation evaluates string operations
func (ee *ExecutionEngine) evaluateStringOperation(left, right *types.Value, operator string) (*types.Value, error) {
	leftStr := left.String
	rightStr := right.String

	switch operator {
	case "+":
		return &types.Value{
			Type:   types.StringType,
			String: leftStr + rightStr,
		}, nil
	case "-":
		// TWX compatibility: string subtraction (numeric operation on string values)
		leftNum := left.ToNumber()
		rightNum := right.ToNumber()
		return &types.Value{
			Type:   types.NumberType,
			Number: leftNum - rightNum,
		}, nil
	case "=":
		result := 0.0
		if leftStr == rightStr {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	case "<>", "!=":
		result := 0.0
		if leftStr != rightStr {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported string operator: %s", operator)
	}
}

// evaluateNumericOperation evaluates numeric operations
func (ee *ExecutionEngine) evaluateNumericOperation(left, right *types.Value, operator string) (*types.Value, error) {
	leftNum := left.ToNumber()
	rightNum := right.ToNumber()

	switch operator {
	case "+":
		return &types.Value{
			Type:   types.NumberType,
			Number: leftNum + rightNum,
		}, nil
	case "-":
		return &types.Value{
			Type:   types.NumberType,
			Number: leftNum - rightNum,
		}, nil
	case "*":
		return &types.Value{
			Type:   types.NumberType,
			Number: leftNum * rightNum,
		}, nil
	case "/":
		if rightNum == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: leftNum / rightNum,
		}, nil
	case "=":
		result := 0.0
		if leftNum == rightNum {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	case "<>", "!=":
		result := 0.0
		if leftNum != rightNum {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	case "<":
		result := 0.0
		if leftNum < rightNum {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	case ">":
		result := 0.0
		if leftNum > rightNum {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	case "<=":
		result := 0.0
		if leftNum <= rightNum {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	case ">=":
		result := 0.0
		if leftNum >= rightNum {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	case "and", "AND":
		result := 0.0
		if leftNum != 0 && rightNum != 0 {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	case "or", "OR":
		result := 0.0
		if leftNum != 0 || rightNum != 0 {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	case "xor", "XOR":
		result := 0.0
		leftTruthy := leftNum != 0
		rightTruthy := rightNum != 0
		if (leftTruthy && !rightTruthy) || (!leftTruthy && rightTruthy) {
			result = 1.0
		}
		return &types.Value{
			Type:   types.NumberType,
			Number: result,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported numeric operator: %s", operator)
	}
}
