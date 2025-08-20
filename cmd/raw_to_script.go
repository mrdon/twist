package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

// LogEntry represents either OnData or SendData
type LogEntry struct {
	Type string // "OnData" or "SendData"
	Size int
	Data string
	Line int // Line number in raw.log
}

// ScriptLine represents a single line in the raw script format
type ScriptLine struct {
	Direction string // "<" for server, ">" for client
	Data      string // the raw data
}

func main() {
	var (
		logFile     = flag.String("log", "raw.log", "Path to raw.log file")
		startLine   = flag.Int("start-line", 1, "Starting line number (1-based)")
		endLine     = flag.Int("end-line", -1, "Ending line number (1-based, -1 for end of file)")
		outputFile  = flag.String("output", "", "Output YAML file path (prints to stdout if not specified)")
		name        = flag.String("name", "Generated Test", "Test name")
		description = flag.String("desc", "Auto-generated test from raw.log", "Test description")
	)
	flag.Parse()

	entries, err := parseRawLogWithRange(*logFile, *startLine, *endLine)
	if err != nil {
		fmt.Printf("Error parsing raw.log: %v\n", err)
		os.Exit(1)
	}

	scriptLines := generateScriptLines(entries)

	if *outputFile != "" {
		err = writeScriptToFile(scriptLines, *outputFile, *name, *description)
		if err != nil {
			fmt.Printf("Error writing script file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated test script: %s\n", *outputFile)
	} else {
		printScript(scriptLines, *name, *description)
	}
}

func parseRawLogWithRange(filename string, startLine, endLine int) ([]LogEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)

	// Regex to match chunk headers
	chunkRegex := regexp.MustCompile(`^(OnData|SendData) chunk \((\d+) bytes\):$`)
	currentLine := 1

	for scanner.Scan() {
		line := scanner.Text()

		// Check if we're in the specified range
		if currentLine < startLine {
			currentLine++
			continue
		}
		if endLine != -1 && currentLine > endLine {
			break
		}

		if matches := chunkRegex.FindStringSubmatch(line); matches != nil {
			entryType := matches[1]
			size, _ := strconv.Atoi(matches[2])

			// Read the next line which contains the data
			if scanner.Scan() {
				currentLine++ // Account for the data line
				if endLine != -1 && currentLine > endLine {
					break
				}
				data := scanner.Text()
				entries = append(entries, LogEntry{
					Type: entryType,
					Size: size,
					Data: data,
					Line: currentLine - 1, // Line of the chunk header
				})
			}
		}
		currentLine++
	}

	return entries, scanner.Err()
}

func generateScriptLines(entries []LogEntry) []ScriptLine {
	var lines []ScriptLine

	// Simply convert each entry to its raw form in order
	for _, entry := range entries {
		if entry.Type == "OnData" {
			// Server sends data
			lines = append(lines, ScriptLine{
				Direction: "<",
				Data:      entry.Data,
			})
		} else if entry.Type == "SendData" {
			// Client sends data
			lines = append(lines, ScriptLine{
				Direction: ">",
				Data:      entry.Data,
			})
		}
	}

	return lines
}

func printScript(lines []ScriptLine, name, description string) {
	fmt.Printf("# %s\n", name)
	fmt.Printf("# %s\n\n", description)

	for _, line := range lines {
		fmt.Printf("%s %s\n", line.Direction, line.Data)
	}
}

func writeScriptToFile(lines []ScriptLine, filename, name, description string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "# %s\n", name)
	fmt.Fprintf(file, "# %s\n\n", description)

	for _, line := range lines {
		fmt.Fprintf(file, "%s %s\n", line.Direction, line.Data)
	}

	return nil
}
