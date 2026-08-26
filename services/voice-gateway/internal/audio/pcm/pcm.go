// Package pcm decodes LiveKit Opus audio tracks into linear16 PCM samples.
package pcm

import (
	"context"
	"encoding/binary"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const defaultLogInterval = 3 * time.Second

// Sink receives resampled PCM16 samples from a remote LiveKit audio track.
type Sink struct {
	label    string
	onPCM    func([]byte)
	logEvery time.Duration

	totalBytes atomic.Uint64
	lastLog    atomic.Int64
	closed     atomic.Bool
}

// NewSink returns a writer that converts PCM16 samples to linear16 bytes.
func NewSink(label string, onPCM func([]byte)) *Sink {
	return &Sink{
		label:    label,
		onPCM:    onPCM,
		logEvery: defaultLogInterval,
	}
}

// WriteSample stores one PCM16 frame as linear16 bytes.
func (s *Sink) WriteSample(sample []int16) error {
	if s.closed.Load() {
		return nil
	}

	pcm := SampleToLinear16(sample)
	if len(pcm) == 0 {
		return nil
	}

	s.totalBytes.Add(uint64(len(pcm)))
	s.maybeLog()
	if s.onPCM != nil {
		s.onPCM(pcm)
	}
	return nil
}

// Close stops the sink and logs totals.
func (s *Sink) Close() error {
	if s.closed.Swap(true) {
		return nil
	}
	log.Printf("pcm: stopped %s total_bytes=%d", s.label, s.totalBytes.Load())
	return nil
}

func (s *Sink) maybeLog() {
	if s.logEvery <= 0 {
		return
	}

	now := time.Now().UnixNano()
	prev := s.lastLog.Load()
	if now-prev < s.logEvery.Nanoseconds() {
		return
	}
	if !s.lastLog.CompareAndSwap(prev, now) {
		return
	}

	log.Printf("pcm: streaming %s bytes=%d", s.label, s.totalBytes.Load())
}

// SampleToLinear16 encodes mono PCM16 samples as little-endian bytes for STT providers.
func SampleToLinear16(sample []int16) []byte {
	if len(sample) == 0 {
		return nil
	}
	out := make([]byte, len(sample)*2)
	for i, v := range sample {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(v))
	}
	return out
}

// RemoteTrack decodes one subscribed Opus track until closed.
type RemoteTrack struct {
	cancel context.CancelFunc
	sink   *Sink
}

// TrackSet tracks active PCM pipelines keyed by LiveKit track SID.
type TrackSet struct {
	mu      sync.Mutex
	entries map[string]*RemoteTrack
}

// NewTrackSet returns an empty pipeline registry.
func NewTrackSet() *TrackSet {
	return &TrackSet{entries: make(map[string]*RemoteTrack)}
}

// Add registers a pipeline for the track SID.
func (s *TrackSet) Add(trackID string, track *RemoteTrack) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[trackID] = track
}

// Remove closes and deletes a pipeline for the track SID.
func (s *TrackSet) Remove(trackID string) {
	s.mu.Lock()
	track := s.entries[trackID]
	delete(s.entries, trackID)
	s.mu.Unlock()
	if track != nil {
		track.Close()
	}
}

// CloseAll stops every registered pipeline.
func (s *TrackSet) CloseAll() {
	s.mu.Lock()
	entries := s.entries
	s.entries = make(map[string]*RemoteTrack)
	s.mu.Unlock()

	for id, track := range entries {
		if track != nil {
			track.Close()
		}
		log.Printf("pcm: closed track=%s", id)
	}
}

// Close stops decoding and flushes sink logging.
func (t *RemoteTrack) Close() {
	if t == nil {
		return
	}
	if t.cancel != nil {
		t.cancel()
	}
	if t.sink != nil {
		_ = t.sink.Close()
	}
}
