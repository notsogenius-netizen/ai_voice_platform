package stt_test

import (
	"context"

	"github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"
)

type fakeStream struct{}

func (fakeStream) WritePCM([]byte) error              { return nil }
func (fakeStream) Transcripts() <-chan stt.Transcript { return nil }
func (fakeStream) Close() error                       { return nil }

type fakeClient struct{}

func (fakeClient) Open(context.Context, stt.Session) (stt.Stream, error) {
	return fakeStream{}, nil
}

var (
	_ stt.Stream = fakeStream{}
	_ stt.Client = fakeClient{}
)
