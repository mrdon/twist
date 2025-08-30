Coding style:
* Try to keep methods under 200 lines of code
* Try to keep files under 500 lines of code

Testing:
* Favor integration tests in integration/ that use a real vm and real test db
* Run 'make test' to run all tests
* Do not try to run the application - ask the user to run it and report results

Debugging:
* Always keep the internal/log package import in all files
* Use structured logging with log.Debug(), log.Info(), log.Warn(), log.Error()
* Use key-value pairs for structured logging: `log.Info("Processing user input", "user", username, "command", cmd)`
* Use log.Debug() for low-level details that can be filtered out in production
* Use log.Info() for important application events and flow tracking
* Use log.Warn() for potential issues or unusual conditions
* Use log.Error() for errors and panic recovery: `defer func() { if r := recover() { log.Error("PANIC in function", "function", "SomeFunction", "error", r) } }()`
* Logging output: defaults to console (tests, utilities), writes to twist_debug.log when main application runs

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

Source of truth:
* The source of truth for the proxy is in twx-src. See docs/twx-arch.md for how that is structured
* Our proxy is aiming for 100% compatability with TWX, so treat any algorithms or patterns in twx as the source of truth
* We don't care about backwards compatability or proxy features that conflict with TWX compatability