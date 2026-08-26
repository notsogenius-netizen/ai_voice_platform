package deepgram

import "github.com/sourabh/ai-voice-platform/services/voice-gateway/internal/stt"

var _ stt.Client = (*Client)(nil)
