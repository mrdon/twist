Coding style:
* Try to keep methods under 200 lines of code
* Try to keep files under 500 lines of code

Testing:
* Favor integration tests in integration/ that use a real vm and real test db
* Run 'make test' to run all tests
* Do not try to run the application - ask the user to run it and report results

Debugging:
* Always keep the internal/debug package import in all files
* Use debug.Log() for debugging during development
* Remove debug.Log() calls before final commits, except for critical error recovery (panics)
* Keep debug.Log() calls in panic recovery blocks: `defer func() { if r := recover() { debug.Log("PANIC: %v", r) } }()`

UI Development (tview):
* NEVER call QueueUpdateDraw() from within another QueueUpdateDraw() callback - this causes deadlocks
* Use goroutines for async UI updates: `go func() { app.QueueUpdateDraw(func() { ... }) }()`
* Handle connection events asynchronously to prevent blocking the main event loop
* Always use non-blocking patterns for event handling to ensure UI responsiveness

Lexer Development (golex):
* NEVER modify generated lexer files (game_lexer_generated.go) - they are auto-generated from .l files
* Only modify the .l lexer definition file (game_lexer.l) and regenerate using golex
* Use `~/go/bin/golex -o game_lexer_generated.go game_lexer.l` to regenerate lexer code
* Debug lexer issues by modifying the .l file patterns and regenerating, not by editing generated code