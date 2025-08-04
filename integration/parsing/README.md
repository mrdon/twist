# Parsing Integration Tests

This directory contains integration tests for the TradeWars 2002 parsing system using real-world game server data.

## Test Files

- `real_world_parsing_test.go` - Main integration tests using actual raw.log data
- `real_world_data.txt` - Real TradeWars 2002 server data copied from raw.log

## Running Tests

### Basic Usage
```bash
# Run all parsing integration tests
go test -tags=integration ./integration/parsing

# Run a specific test
go test -tags=integration ./integration/parsing -run TestRealWorldParsingChunked
```

### Random Chunking Tests

The chunked parsing tests simulate realistic network conditions by feeding data in random-sized chunks. This helps identify parsing issues that might occur when data arrives in unpredictable packet sizes.

#### Reproducible Test Runs

When a test fails or you want to reproduce a specific chunking pattern, use the `CHUNK_SEED` environment variable:

```bash
# The test will print a seed like this:
# Generated random chunking seed: 1722693845123456789
# To reproduce this exact chunking pattern, set: CHUNK_SEED=1722693845123456789

# Reproduce the exact same chunking pattern:
CHUNK_SEED=1722693845123456789 go test -tags=integration ./integration/parsing -run TestRealWorldParsingChunked

# Use a specific seed for consistent testing:
CHUNK_SEED=12345 go test -tags=integration ./integration/parsing -run TestRealWorldParsingChunked
```

#### Debugging Parsing Issues

1. **Run the test normally** to see if it passes
2. **If it fails**, note the seed from the log output
3. **Re-run with that specific seed** to reproduce the exact failure
4. **Examine the chunk boundaries** in the log output to identify where parsing broke
5. **Use smaller seeds** (like `CHUNK_SEED=1`) to force very small chunks for stress testing

### Test Types

- **TestRealWorldParsingChunked**: Random chunk sizes with controllable seed
- **TestRealWorldParsingLineByLine**: Systematic line-by-line processing  
- **TestRealWorldParsingByteStream**: Complete data as single stream
- **TestRealWorldParsingWithKnownSeed**: Uses a specific seed (12345) for consistent small chunks

## What The Tests Validate

1. **Sector Detection**: Verifies "Sector  : 2142" is correctly parsed
2. **Port Parsing**: Validates "Trading" headers and product lines are detected
3. **Command Prompts**: Ensures command prompts with timing are recognized
4. **Token Processing**: Confirms the lexer processes all patterns correctly
5. **Database Integration**: Checks that parsed data is stored correctly

## Network Simulation

The chunked tests simulate real-world network conditions:
- **Random chunk sizes**: 1-50 bytes to mimic variable packet sizes
- **Network delays**: Random 1-5ms delays (10% chance) to simulate latency
- **Boundary conditions**: Data can be split at any byte boundary, including:
  - Middle of ANSI escape sequences
  - Middle of pattern matches
  - Middle of multi-byte UTF-8 characters

This helps ensure the streaming lexer can handle data arriving in any possible fragmentation pattern.