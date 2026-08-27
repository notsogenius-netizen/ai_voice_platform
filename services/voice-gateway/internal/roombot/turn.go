package roombot

import (
	"context"
	"errors"
	"log"
	"strings"

	lksdk "github.com/livekit/server-sdk-go/v2"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/orchestrator"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/tts"
)

type speakState struct {
	ignored bool
	spoke   bool
}

func (p *turnPipeline) handleFinal(ctx context.Context, tr stt.Transcript) {
	if p == nil || p.orch == nil {
		return
	}
	p.lockTurn()
	defer p.unlockTurn()
	p.interruptPlayback()

	var buf sentenceBuffer
	state, reply, err := p.streamReply(ctx, tr, &buf)
	if err != nil {
		p.logTurnError(tr, err)
		return
	}
	if state.ignored {
		return
	}
	logForwarded(tr)
	p.afterStream(ctx, tr, &buf, &state, reply.Text)
}

func (p *turnPipeline) lockTurn() {
	if p.turnMu != nil {
		p.turnMu.Lock()
	}
}

func (p *turnPipeline) unlockTurn() {
	if p.turnMu != nil {
		p.turnMu.Unlock()
	}
}

func (p *turnPipeline) interruptPlayback() {
	if p.playback != nil {
		p.playback.Interrupt()
	}
}

func (p *turnPipeline) streamReply(
	ctx context.Context,
	tr stt.Transcript,
	buf *sentenceBuffer,
) (speakState, orchestrator.Reply, error) {
	state := speakState{}
	speak := p.makeSpeaker(ctx, tr.Session.Room, &state)
	reply, err := p.orch.StreamTurn(ctx, orchestrator.Turn{
		SessionID: tr.Session.Room,
		Text:      tr.Text,
		IsFinal:   true,
	}, func(chunk string) error {
		return speakSentences(speak, buf.Push(chunk))
	})
	return state, reply, err
}

func (p *turnPipeline) afterStream(
	ctx context.Context,
	tr stt.Transcript,
	buf *sentenceBuffer,
	state *speakState,
	fullText string,
) {
	speak := p.makeSpeaker(ctx, tr.Session.Room, state)
	if rest := buf.Flush(); rest != "" {
		if err := speak(rest); err != nil {
			p.logSpeakError(tr.Session.Room, err)
			return
		}
	}
	if state.spoke || fullText == "" {
		return
	}
	if err := speak(fullText); err != nil {
		p.logSpeakError(tr.Session.Room, err)
	}
}

func (p *turnPipeline) makeSpeaker(
	ctx context.Context,
	roomName string,
	state *speakState,
) func(string) error {
	return func(text string) error {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil
		}
		if err := p.speakSentence(ctx, roomName, text); err != nil {
			return err
		}
		state.spoke = true
		return nil
	}
}

func speakSentences(speak func(string) error, sentences []string) error {
	for _, sentence := range sentences {
		if err := speak(sentence); err != nil {
			return err
		}
	}
	return nil
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
	return p.publishAudio(ctx, roomName, audio.Ogg)
}

func (p *turnPipeline) publishAudio(ctx context.Context, roomName string, ogg []byte) error {
	room := p.currentRoom()
	if room == nil || p.playback == nil {
		log.Printf("tts: skipped_publish room=%s", roomName)
		return nil
	}
	return p.playback.PlayOgg(ctx, room, ogg)
}

func (p *turnPipeline) currentRoom() *lksdk.Room {
	if p.room == nil {
		return nil
	}
	return p.room()
}

func (p *turnPipeline) logTurnError(tr stt.Transcript, err error) {
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
	p.interruptPlayback()
}

func (p *turnPipeline) logSpeakError(roomName string, err error) {
	if errors.Is(err, errReplyInterrupted) {
		log.Printf("roombot: reply interrupted room=%s", roomName)
		return
	}
	log.Printf("tts: speak failed room=%s: %v", roomName, err)
	p.interruptPlayback()
}

func logForwarded(tr stt.Transcript) {
	log.Printf(
		"orchestrator: forwarded turn room=%s identity=%s track=%s",
		tr.Session.Room,
		tr.Session.Participant,
		tr.Session.TrackID,
	)
}
