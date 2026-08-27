package roombot

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/tts"
)

type sttPipe struct {
	stream stt.Stream
	label  string
	dead   atomic.Bool
	once   sync.Once
}

// turnPipeline forwards finals to the orchestrator and speaks replies.
type turnPipeline struct {
	orch     orchestrator.Client
	tts      tts.Client
	room     func() *lksdk.Room
	playback *replyPlayback
	turnMu   *sync.Mutex
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

func readTranscripts(ctx context.Context, pipe *sttPipe, pipeline *turnPipeline) {
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
			if pipeline == nil {
				continue
			}
			if pipeline.playback != nil && pipeline.playback.Playing() && strings.TrimSpace(tr.Text) != "" {
				log.Printf(
					"roombot: barge-in room=%s identity=%s track=%s",
					tr.Session.Room,
					tr.Session.Participant,
					tr.Session.TrackID,
				)
				pipeline.playback.Interrupt()
			}
			if pipeline.orch != nil && tr.IsFinal {
				go pipeline.handleFinal(ctx, tr)
			}
		}
	}
}

func (p *turnPipeline) handleFinal(ctx context.Context, tr stt.Transcript) {
	if p == nil || p.orch == nil {
		return
	}
	if p.turnMu != nil {
		p.turnMu.Lock()
		defer p.turnMu.Unlock()
	}
	if p.playback != nil {
		p.playback.Interrupt()
	}

	var buf sentenceBuffer
	spoke := false
	speak := func(text string) error {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil
		}
		if err := p.speakSentence(ctx, tr.Session.Room, text); err != nil {
			return err
		}
		spoke = true
		return nil
	}

	reply, err := p.orch.StreamTurn(ctx, orchestrator.Turn{
		SessionID: tr.Session.Room,
		Text:      tr.Text,
		IsFinal:   true,
	}, func(chunk string) error {
		for _, sentence := range buf.Push(chunk) {
			if err := speak(sentence); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errReplyInterrupted) {
			log.Printf(
				"roombot: reply interrupted room=%s identity=%s track=%s",
				tr.Session.Room,
				tr.Session.Participant,
				tr.Session.TrackID,
			)
			return
		}
		log.Printf(
			"orchestrator: turn failed room=%s identity=%s track=%s: %v",
			tr.Session.Room,
			tr.Session.Participant,
			tr.Session.TrackID,
			err,
		)
		if p.playback != nil {
			p.playback.Interrupt()
		}
		return
	}
	if reply.Ignored {
		return
	}
	log.Printf(
		"orchestrator: forwarded turn room=%s identity=%s track=%s",
		tr.Session.Room,
		tr.Session.Participant,
		tr.Session.TrackID,
	)

	if rest := buf.Flush(); rest != "" {
		if err := speak(rest); err != nil {
			if errors.Is(err, errReplyInterrupted) {
				log.Printf("roombot: reply interrupted room=%s", tr.Session.Room)
				return
			}
			log.Printf("tts: speak failed room=%s: %v", tr.Session.Room, err)
			if p.playback != nil {
				p.playback.Interrupt()
			}
			return
		}
	}

	if !spoke && reply.Text != "" {
		if err := speak(reply.Text); err != nil {
			if errors.Is(err, errReplyInterrupted) {
				log.Printf("roombot: reply interrupted room=%s", tr.Session.Room)
				return
			}
			log.Printf("tts: speak failed room=%s: %v", tr.Session.Room, err)
			if p.playback != nil {
				p.playback.Interrupt()
			}
			return
		}
	}
}

func (p *turnPipeline) speakSentence(ctx context.Context, roomName, text string) error {
	text = strings.TrimSpace(text)
	if text == "" || p.tts == nil {
		return nil
	}

	audio, err := p.tts.Synthesize(ctx, tts.Request{Text: text})
	if err != nil {
		return err
	}
	log.Printf("tts: synthesized room=%s bytes=%d", roomName, len(audio.Ogg))

	room := (*lksdk.Room)(nil)
	if p.room != nil {
		room = p.room()
	}
	if room == nil || p.playback == nil {
		log.Printf("tts: skipped_publish room=%s", roomName)
		return nil
	}
	return p.playback.PlayOgg(ctx, room, audio.Ogg)
}

// forwardFinalTranscript is retained for focused unit tests of orchestrator forwarding.
func forwardFinalTranscript(ctx context.Context, orch orchestrator.Client, tr stt.Transcript) {
	pipeline := &turnPipeline{orch: orch}
	pipeline.handleFinal(ctx, tr)
}
