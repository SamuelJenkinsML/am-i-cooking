package calc

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/SamuelJenkinsML/am-i-cooking/internal/parser"
)

// Metrics holds all computed burn rate metrics.
type Metrics struct {
	CurrentRate  float64 // tokens/min over last 5 min
	SustainedRate float64 // tokens/min over last 30 min
	OverallRate  float64 // tokens/min over entire window

	TotalWeightedTokens float64
	TotalRawTokens      int

	WindowElapsed   time.Duration
	WindowRemaining time.Duration
	WindowSize      time.Duration

	EstimatedCost float64 // £ estimate

	OpusPercent  float64
	HaikuPercent float64
	SonnetPercent float64

	Verdict     string
	GaugePercent float64 // 0-100 for the gauge
}

// WeightedTokens computes a cost-weighted token count for a record.
// Weights reflect relative pricing: output 5x, cache creation 1.25x, cache read 0.1x, input 1x.
func WeightedTokens(r parser.UsageRecord) float64 {
	return float64(r.InputTokens)*1.0 +
		float64(r.OutputTokens)*5.0 +
		float64(r.CacheCreationTokens)*1.25 +
		float64(r.CacheReadTokens)*0.1
}

// RawTokens returns the total raw token count for a record.
func RawTokens(r parser.UsageRecord) int {
	return r.InputTokens + r.OutputTokens + r.CacheCreationTokens + r.CacheReadTokens
}

// Calculate computes all metrics from a set of usage records.
func Calculate(records []parser.UsageRecord, windowSize time.Duration) Metrics {
	now := time.Now()
	windowStart := now.Add(-windowSize)

	m := Metrics{
		WindowSize: windowSize,
	}

	if len(records) == 0 {
		m.WindowElapsed = 0
		m.WindowRemaining = windowSize
		m.Verdict = "stone cold... get cooking!"
		m.GaugePercent = 0
		return m
	}

	// Sort by timestamp
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.Before(records[j].Timestamp)
	})

	// Filter to window
	var inWindow []parser.UsageRecord
	for _, r := range records {
		if !r.Timestamp.Before(windowStart) {
			inWindow = append(inWindow, r)
		}
	}

	if len(inWindow) == 0 {
		m.WindowElapsed = 0
		m.WindowRemaining = windowSize
		m.Verdict = "stone cold... get cooking!"
		m.GaugePercent = 0
		return m
	}

	// Window timing based on first record
	earliest := inWindow[0].Timestamp
	m.WindowElapsed = now.Sub(earliest)
	if m.WindowElapsed > windowSize {
		m.WindowElapsed = windowSize
	}
	m.WindowRemaining = windowSize - m.WindowElapsed
	if m.WindowRemaining < 0 {
		m.WindowRemaining = 0
	}

	// Accumulate totals and per-bucket rates
	var totalWeighted float64
	var recent5Weighted float64
	var recent30Weighted float64
	var opusTokens, haikuTokens, sonnetTokens float64

	cutoff5 := now.Add(-5 * time.Minute)
	cutoff30 := now.Add(-30 * time.Minute)

	for _, r := range inWindow {
		w := WeightedTokens(r)
		totalWeighted += w
		m.TotalRawTokens += RawTokens(r)

		if !r.Timestamp.Before(cutoff5) {
			recent5Weighted += w
		}
		if !r.Timestamp.Before(cutoff30) {
			recent30Weighted += w
		}

		model := strings.ToLower(r.Model)
		switch {
		case strings.Contains(model, "opus"):
			opusTokens += w
		case strings.Contains(model, "haiku"):
			haikuTokens += w
		default:
			sonnetTokens += w
		}
	}

	m.TotalWeightedTokens = totalWeighted

	// Rates — use actual time span of records in each bucket, not the full bucket duration.
	// This prevents a single response from being spread across an artificially long window.
	// Minimum 1 minute to avoid extreme spikes from a single recent response.
	minDuration := 1.0 // minutes

	elapsed5 := bucketSpan(inWindow, cutoff5, now, minDuration)
	elapsed30 := bucketSpan(inWindow, cutoff30, now, minDuration)
	elapsedAll := m.WindowElapsed.Minutes()
	if elapsedAll < minDuration {
		elapsedAll = minDuration
	}

	if elapsed5 > 0 {
		m.CurrentRate = recent5Weighted / elapsed5
	}
	if elapsed30 > 0 {
		m.SustainedRate = recent30Weighted / elapsed30
	}
	if elapsedAll > 0 {
		m.OverallRate = totalWeighted / elapsedAll
	}

	// Model breakdown
	if totalWeighted > 0 {
		m.OpusPercent = (opusTokens / totalWeighted) * 100
		m.HaikuPercent = (haikuTokens / totalWeighted) * 100
		m.SonnetPercent = (sonnetTokens / totalWeighted) * 100
	}

	// Cost estimate: rough approximation based on weighted tokens as fraction of budget
	// Assume ~£90/month, ~720 hours/month → ~£0.625/5h window
	// A very active 5h window might use ~10M weighted tokens
	// So rough: cost = (weightedTokens / 10_000_000) * £0.625
	// This is intentionally approximate
	m.EstimatedCost = (totalWeighted / 10_000_000.0) * 0.625

	// Verdict based on current rate (tokens/min)
	m.Verdict, m.GaugePercent = verdict(m.CurrentRate)

	return m
}

// bucketSpan returns the time span in minutes from the earliest record after cutoff to now,
// clamped to at least minMinutes. If no records fall in the bucket, returns 0.
func bucketSpan(records []parser.UsageRecord, cutoff, now time.Time, minMinutes float64) float64 {
	var earliest time.Time
	found := false
	for _, r := range records {
		if !r.Timestamp.Before(cutoff) {
			if !found || r.Timestamp.Before(earliest) {
				earliest = r.Timestamp
				found = true
			}
		}
	}
	if !found {
		return 0
	}
	span := now.Sub(earliest).Minutes()
	if span < minMinutes {
		span = minMinutes
	}
	return span
}

func verdict(tokPerMin float64) (string, float64) {
	pct := rateToPercent(tokPerMin)

	var v string
	switch {
	case tokPerMin < 1000:
		v = "lukewarm... pick up the pace!"
	case tokPerMin < 10000:
		v = "simmering nicely"
	case tokPerMin < 100000:
		v = "NOW you're cooking!"
	default:
		v = "ABSOLUTELY COOKING"
	}

	return v, pct
}

// rateToPercent maps burn rate to a 0-100 gauge on a log scale.
// Calibrated for real Opus usage where cache tokens dominate:
//   1,000 tok/min → 25% (light usage)
//  10,000 tok/min → 50% (moderate)
// 100,000 tok/min → 75% (heavy)
// 1,000,000 tok/min → 100% (absolutely cooking)
func rateToPercent(tokPerMin float64) float64 {
	if tokPerMin <= 0 {
		return 0
	}

	// log10 scale: 3 (1k) → 25%, 4 (10k) → 50%, 5 (100k) → 75%, 6 (1M) → 100%
	logVal := math.Log10(tokPerMin)

	// Map [3, 6] → [25, 100]
	pct := ((logVal - 3.0) / 3.0) * 75.0 + 25.0

	// Below 1k: linear ramp from 0 to 25%
	if logVal < 3.0 {
		pct = (tokPerMin / 1000.0) * 25.0
	}

	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	return pct
}
