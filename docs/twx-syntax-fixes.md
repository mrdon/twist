# TWX Syntax Fixes for Integration Tests

## Overview

This document outlines systematic fixes needed to align integration test scripts with authentic TWX script syntax. The current test scripts contain syntax deviations that must be corrected to ensure accurate testing of the TWX compatibility layer.

## Root Cause Analysis

The integration tests were written using Go-inspired syntax patterns rather than authentic TWX script syntax. After comprehensive analysis of:
- Original TWX scripts in `twx-scripts/`  
- TWX Pascal engine implementation in `twx-src/ScriptCmp.pas`
- Current integration test scripts in `integration/scripting/`

Multiple syntax discrepancies have been identified that prevent accurate compatibility testing.

---

## Phase 1: Critical Variable Assignment Fixes ✅ COMPLETED

**Priority:** CRITICAL  
**Estimated Impact:** ~80% of test scripts  
**Complexity:** Medium (requires careful pattern matching)  
**Status:** ✅ **COMPLETED** - January 2025

### Problem
Integration tests use Go-style assignment syntax which is not valid TWX:
```go
// INCORRECT - Not valid TWX syntax
$variable := "value"
$counter := 1
$result := 42.5
```

### Required Fix
Replace with authentic TWX `setVar` command syntax:
```twx
// CORRECT - Authentic TWX syntax
setVar $variable "value"
setVar $counter 1
setVar $result 42.5
```

### Files Requiring Updates
- `integration/scripting/variables_test.go` - Extensive usage throughout
- `integration/scripting/control_test.go` - Lines 15, 62, 118, and many others
- `integration/scripting/array_variables_test.go` - Array element assignments
- `integration/scripting/realistic_scenarios_test.go` - Lines 120, 223, mixed usage
- `integration/scripting/workflows_test.go` - Variable initialization
- `integration/scripting/datetime_test.go` - Time variable assignments
- `integration/scripting/game_test.go` - Game state variables
- `integration/scripting/include_test.go` - Variable assignments in included scenarios
- `integration/scripting/pascal_array_validation_test.go` - Array assignments

### Implementation Strategy
1. **Pattern Search:** Use regex to find all instances of `\$\w+\s*:=`
2. **Context Validation:** Ensure matches are actual assignments, not in comments
3. **Replacement Logic:** 
   - `$var := "string"` → `setVar $var "string"`
   - `$var := 123` → `setVar $var 123`
   - `$var := $other_var` → `setVar $var $other_var`
4. **Verification:** Run syntax validation after each file update

### Testing After Changes
- Ensure all affected tests still pass
- Verify no syntax errors introduced
- Check that variable values are correctly set and retrieved

---

## Phase 2: Arithmetic Command Parameter Fixes ✅ COMPLETED

**Priority:** HIGH  
**Estimated Impact:** Minimal - Only 1 incorrect usage found  
**Complexity:** Low (straightforward parameter removal)  
**Status:** ✅ **COMPLETED** - January 2025

### Problem
Some tests use 3-parameter arithmetic commands, but TWX only supports 2-parameter (in-place modification):
```go
// INCORRECT - 3-parameter form not supported in TWX
add $main_var 100 $main_var
multiply $factorial_result $counter $factorial_result
subtract $counter 1 $counter
divide $amount 2 $amount
```

### Required Fix
Use authentic TWX 2-parameter syntax (in-place modification):
```twx
// CORRECT - TWX 2-parameter form
add $main_var 100
multiply $factorial_result $counter  
subtract $counter 1
divide $amount 2
```

### Implementation Results
**Comprehensive search revealed minimal impact:**
- **Total files searched:** All `integration/scripting/*.go` files
- **Issues found:** Only 1 incorrect usage
- **Files updated:** `integration/scripting/types_test.go`

### Changes Made
**Fixed line 151 in `integration/scripting/types_test.go`:**
```diff
- script: "divide 10 0 $result",  // INCORRECT: 3-parameter syntax
+ script: "setVar $result 10\ndivide $result 0",  // CORRECT: TWX 2-parameter syntax
```

### Validation Confirmed
✅ **Cross-referenced with authentic TWX scripts** in `twx-scripts/`:
- All `divide` commands use 2-parameter form: `divide $variable value`
- Found 30+ examples in real TWX scripts confirming correct syntax
- Mathematical command implementations in `internal/scripting/vm/commands/math.go` verified

✅ **All integration tests pass** including error handling tests

### Implementation Strategy Used
1. ✅ **Search Pattern:** `(add|subtract|multiply|divide)\s+\$\w+\s+[^$\s]+\s+\$\w+` - Found 0 matches
2. ✅ **Broader Search:** `(add|subtract|multiply|divide)\s+\d+\s+\d+\s+\$` - Found 1 match  
3. ✅ **Context Validation:** Verified it was error test case for division by zero
4. ✅ **Fix Applied:** Converted to proper TWX 2-parameter in-place modification syntax

### Verification Reference
✅ **Confirmed against real TWX scripts** in `twx-scripts/` - ALL arithmetic operations use 2-parameter form:
```twx
add $i 1
add $weight[$i] $density[$i]
multiply $experience 2
divide $price 100
```

---

## Phase 3: Command Case Standardization ✅ COMPLETED

**Priority:** MEDIUM  
**Estimated Impact:** ~20% of commands  
**Complexity:** Low (case correction)  
**Status:** ✅ **COMPLETED** - January 2025

### Problem
Inconsistent command casing throughout tests:
```go
// Mixed casing found
setvar $variable "value"  // Should be setVar
cuttext $line $result 1 5  // Should be cutText  
getword $line $word 2      // Should be getWord
setarray $arr 10           // Should be setArray
```

### Required Fix
Standardize to Pascal case as defined in TWX engine:
```twx
// CORRECT - Pascal case from ScriptCmp.pas
setVar $variable "value"
cutText $line $result 1 5
getWord $line $word 2
setArray $arr 10
```

### Command Case Reference
From `twx-src/ScriptCmp.pas` and `twx-scripts/` analysis:
- `setVar` (not `setvar`)
- `cutText` (not `cuttext`)
- `getWord` (not `getword`)
- `setArray` (not `setarray`)
- `saveVar` / `loadVar` (verify against actual engine)
- `killTrigger` (not `killtrigger`)
- `setTextTrigger` (not `settexttrigger`)

### Implementation Results
**Comprehensive standardization completed:**
- **Total commands updated:** 8 primary command types
- **Files modified:** All `integration/scripting/*.go` files  
- **Commands standardized:**
  - `savevar` → `saveVar`
  - `loadvar` → `loadVar`
  - `setvar` → `setVar`
  - `cuttext` → `cutText`
  - `getword` → `getWord`
  - `setarray` → `setArray`
  - `killtrigger` → `killTrigger`
  - `settexttrigger` → `setTextTrigger`

### Implementation Strategy Used
1. ✅ **Command Map Created:** Built comprehensive list from engine registrations
2. ✅ **Systematic Search:** Found all incorrect casing instances across test files
3. ✅ **Bulk Replacement:** Used MultiEdit for efficient case corrections
4. ✅ **Context Preserved:** Maintained all syntax and parameters correctly

### Validation Confirmed
✅ **Engine compatibility verified:** Commands use case-insensitive matching via `strings.ToUpper()`
✅ **All integration tests pass** with standardized Pascal case
✅ **No functionality regression** - identical behavior to previous versions

---

## Phase 4: Syntax Validation and Edge Cases ✅ COMPLETED

**Priority:** LOW-MEDIUM  
**Estimated Impact:** <5% of scripts  
**Complexity:** Medium (requires domain knowledge)  
**Status:** ✅ **COMPLETED** - January 2025

### Issues to Address

#### 4.1 String Literal Handling
Verify proper escaping and quote handling:
```twx
# Ensure proper quote escaping
setVar $message "He said \"Hello\""
# Multi-line string handling with asterisks
setVar $multiline "Line 1*Line 2*Line 3"
```

#### 4.2 Operator Syntax
Verify comparison and logical operators match TWX engine:
```twx
# Correct TWX operators from ScriptCmp.pas
if ($value >= 10)  # OP_GREATEREQUAL
if ($value <= 5)   # OP_LESSEREQUAL  
if ($value <> 0)   # OP_NOTEQUAL
if ($flag AND $other) # OP_AND
```

#### 4.3 Array Index Validation
Ensure 1-based indexing throughout:
```twx
# TWX uses 1-based arrays (Pascal style)
setArray $sectors 100
setVar $sectors[1] "first_sector"  # Index 1, not 0
```

#### 4.4 Label and Control Flow
Verify proper label syntax and control structures:
```twx
:proper_label_name
goto proper_label_name
gosub subroutine_name
# Ensure no invalid characters in labels
```

### Implementation Strategy
1. **Comprehensive Scan:** Review all test scripts for edge cases
2. **Cross-Reference:** Compare against `twx-scripts/` examples
3. **Engine Validation:** Verify against `ScriptCmp.pas` implementation
4. **Test Coverage:** Ensure edge cases are properly tested

---

## Phase 5: Integration and Validation

**Priority:** CRITICAL (Final Phase)  
**Estimated Impact:** All test files  
**Complexity:** High (comprehensive testing)

### Validation Tasks

#### 5.1 Syntax Verification
- Run all integration tests to ensure they pass
- Verify no TWX syntax errors are introduced
- Check that script behavior remains identical

#### 5.2 Cross-Reference Validation
- Compare test script patterns against real TWX scripts in `twx-scripts/`
- Ensure all command usage matches authentic TWX patterns
- Validate complex scenarios mirror real-world TWX script usage

#### 5.3 Engine Compatibility
- Verify compatibility with Pascal engine implementation patterns
- Ensure all syntax aligns with `ScriptCmp.pas` parsing rules
- Test edge cases that might reveal remaining syntax issues

#### 5.4 Documentation Updates
- Update test documentation to reflect correct TWX syntax
- Add comments explaining TWX-specific syntax choices
- Document any remaining intentional deviations (if any)

### Final Verification Checklist
- [x] All `$var := value` replaced with `setVar $var value` ✅ **COMPLETED**
- [x] All arithmetic commands use 2-parameter form ✅ **COMPLETED**
- [x] All commands use correct Pascal case ✅ **COMPLETED**
- [x] All tests pass with correct TWX syntax ✅ **VERIFIED**
- [x] Test behavior is identical to previous versions ✅ **VERIFIED**
- [x] Complex scenarios accurately reflect real TWX script patterns ✅ **VERIFIED**
- [x] No syntax errors when validated against TWX engine rules ✅ **VERIFIED**

---

## Implementation Notes

### Tools and Utilities
- Use regex-based search and replace for bulk changes
- Implement validation scripts to check syntax compliance
- Create test runners to verify behavior preservation

### Risk Mitigation
- Make changes incrementally by phase
- Run tests after each phase completion
- Maintain backup of original test files
- Document any behavior changes discovered

### Success Criteria
- All integration tests use authentic TWX script syntax
- Tests accurately validate Go implementation compatibility with real TWX scripts
- No functionality regression in test coverage
- Test scripts serve as accurate reference for TWX syntax implementation

---

## Context for Future Agents

### Key Reference Files
- `twx-scripts/` - Authentic TWX script examples for syntax reference
- `twx-src/ScriptCmp.pas` - Pascal engine implementation defining TWX syntax rules
- `integration/scripting/` - Test files requiring syntax corrections

### Critical Understanding
- TWX uses Pascal-style syntax with specific command cases
- Variable assignment MUST use `setVar` command, not `:=` operator
- Arithmetic operations are in-place (2-parameter) not 3-parameter
- Array indexing is 1-based (Pascal style)
- Commands use Pascal case conventions

### Validation Approach
Each phase should be completed independently with full testing before proceeding to the next phase. The syntax changes must preserve existing test behavior while ensuring authentic TWX compatibility.