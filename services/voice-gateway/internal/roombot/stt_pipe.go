package roombot

import (
	"context"
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
)

type sttPipe struct {
	stream stt.Stream
	label  string
	dead   atomic.Bool
	once   sync.Once
}

func newSTTPipe(stream stt.Stream, label string) *sttPipe {
	if stream == nil {
		return nil
	}
	return &sttPipe{stream: stream, label: label}
}

func (p *sttPipe) write(pcm []byte) {
	if p == nil || p.dead.Load() {
		return
	}
	if err := p.stream.WritePCM(pcm); err != nil {
		p.end(err)
	}
}

func (p *sttPipe) writeFn() func([]byte) {
	if p == nil {
		return nil
	}
	return p.write
}

func (p *sttPipe) end(reason error) {
	if p == nil {
		return
	}
	p.once.Do(func() {
		p.dead.Store(true)
		if reason != nil {
			log.Printf("roombot: stt ended %s: %v", p.label, reason)
		}
		_ = p.stream.Close()
	})
}

func (p *sttPipe) closeQuietly() {
	if p == nil {
		return
	}
	p.once.Do(func() {
		p.dead.Store(true)
		_ = p.stream.Close()
	})
}

func readTranscripts(ctx context.Context, pipe *sttPipe) {
	if pipe == nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case tr, ok := <-pipe.stream.Transcripts():
			if !ok {
				pipe.end(errors.New("transcript channel closed"))
				return
			}
			logTranscript(tr)
		}
	}
}
