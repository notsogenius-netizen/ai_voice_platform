package pcm

// Resample converts mono PCM16 samples to a new sample rate using linear interpolation.
func Resample(dstRate, srcRate int, src []int16) []int16 {
	if len(src) == 0 || dstRate <= 0 || srcRate <= 0 || dstRate == srcRate {
		out := make([]int16, len(src))
		copy(out, src)
		return out
	}

	outLen := len(src) * dstRate / srcRate
	if outLen == 0 {
		return nil
	}

	out := make([]int16, outLen)
	for i := range out {
		pos := float64(i) * float64(srcRate) / float64(dstRate)
		idx := int(pos)
		if idx >= len(src)-1 {
			out[i] = src[len(src)-1]
			continue
		}
		frac := pos - float64(idx)
		a := float64(src[idx])
		b := float64(src[idx+1])
		out[i] = int16(a + frac*(b-a))
	}
	return out
}
