

package parsing

import (
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
	"twist/internal/proxy/streaming"
)

// RandomDataFeeder simulates network conditions by feeding data in random-sized chunks
type RandomDataFeeder struct {
	t    *testing.T
	seed int64
}

// NewRandomDataFeeder creates a new random data feeder
func NewRandomDataFeeder(t *testing.T) *RandomDataFeeder {
	feeder := &RandomDataFeeder{t: t}
	feeder.initSeed()
	return feeder
}

// initSeed initializes the random seed from environment or generates a new one
func (f *RandomDataFeeder) initSeed() {
	if seedStr := os.Getenv("CHUNK_SEED"); seedStr != "" {
		var err error
		f.seed, err = strconv.ParseInt(seedStr, 10, 64)
		if err != nil {
			f.t.Logf("Warning: Invalid CHUNK_SEED '%s', using random seed", seedStr)
			f.seed = time.Now().UnixNano()
		}
		f.t.Logf("Using chunking seed from CHUNK_SEED environment variable: %d", f.seed)
	} else {
		f.seed = time.Now().UnixNano()
		f.t.Logf("Generated random chunking seed: %d", f.seed)
		f.t.Logf("To reproduce this exact chunking pattern, set: CHUNK_SEED=%d", f.seed)
	}
}

// FeedData feeds data to the parser in random chunks
func (f *RandomDataFeeder) FeedData(twxParser *streaming.TWXParser, data []byte) {
	rand.Seed(f.seed)
	
	dataLen := len(data)
	processed := 0
	chunkCount := 0

	f.t.Logf("Feeding %d bytes in random chunks...", dataLen)

	for processed < dataLen {
		// Random chunk size between 1 and 50 bytes
		chunkSize := rand.Intn(50) + 1
		if processed+chunkSize > dataLen {
			chunkSize = dataLen - processed
		}

		chunk := data[processed : processed+chunkSize]
		chunkCount++
		
		// Log chunk info (truncate if too long for readability)
		chunkStr := string(chunk)
		if len(chunkStr) > 40 {
			chunkStr = chunkStr[:37] + "..."
		}
		f.t.Logf("Chunk %d (size=%d, range=%d-%d): %q", 
			chunkCount, chunkSize, processed, processed+chunkSize-1, chunkStr)
		
		// Process chunk
		twxParser.ProcessChunk(chunk)
		
		processed += chunkSize
		
		// Simulate network delay occasionally (10% chance)
		if rand.Intn(10) == 0 {
			delayMs := rand.Intn(5) + 1 // 1-5ms delay
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
			f.t.Logf("  -> Simulated network delay: %dms", delayMs)
		}
	}

	f.t.Logf("Completed: Fed %d chunks totaling %d bytes (seed: %d)", chunkCount, dataLen, f.seed)
}