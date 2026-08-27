// Package deepgram implements TTS over the Deepgram Speak REST API.
package deepgram

import "github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/tts"

var _ tts.Client = (*Client)(nil)
