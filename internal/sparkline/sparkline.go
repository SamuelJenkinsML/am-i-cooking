package sparkline

// blocks maps values 0-8 to Unicode block characters.
var blocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Buffer stores a rolling window of rate samples.
type Buffer struct {
	samples []float64
	max     int
}

// NewBuffer creates a buffer with the given capacity.
func NewBuffer(maxSamples int) *Buffer {
	return &Buffer{max: maxSamples}
}

// Add appends a sample, dropping the oldest if at capacity.
func (b *Buffer) Add(rate float64) {
	b.samples = append(b.samples, rate)
	if len(b.samples) > b.max {
		b.samples = b.samples[len(b.samples)-b.max:]
	}
}

// Samples returns the current samples.
func (b *Buffer) Samples() []float64 {
	return b.samples
}

// Render draws a sparkline from samples at the given width.
// If there are more samples than width, adjacent samples are averaged.
// If there are fewer, the output is left-padded with spaces.
func Render(samples []float64, width int) string {
	if width <= 0 {
		return ""
	}

	// Build display values at target width
	display := make([]float64, width)

	if len(samples) == 0 {
		// All spaces
		runes := make([]rune, width)
		for i := range runes {
			runes[i] = ' '
		}
		return string(runes)
	}

	if len(samples) <= width {
		// Left-pad with zeros, then copy samples
		offset := width - len(samples)
		for i, v := range samples {
			display[offset+i] = v
		}
	} else {
		// Downsample by averaging
		ratio := float64(len(samples)) / float64(width)
		for i := 0; i < width; i++ {
			start := int(float64(i) * ratio)
			end := int(float64(i+1) * ratio)
			if end > len(samples) {
				end = len(samples)
			}
			sum := 0.0
			count := 0
			for j := start; j < end; j++ {
				sum += samples[j]
				count++
			}
			if count > 0 {
				display[i] = sum / float64(count)
			}
		}
	}

	// Find max for scaling
	maxVal := 0.0
	for _, v := range display {
		if v > maxVal {
			maxVal = v
		}
	}

	// Map to block characters
	runes := make([]rune, width)
	for i, v := range display {
		if maxVal <= 0 {
			runes[i] = ' '
		} else {
			idx := int((v / maxVal) * 8)
			if idx > 8 {
				idx = 8
			}
			runes[i] = blocks[idx]
		}
	}

	return string(runes)
}
