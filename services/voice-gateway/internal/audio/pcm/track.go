package pcm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	gopus "github.com/thesyncim/gopus"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
)

const opusSampleRate = 48000

// StartRemoteTrack decodes track audio to linear16 PCM at the target sample rate.
func StartRemoteTrack(
	ctx context.Context,
	track *webrtc.TrackRemote,
	sampleRate int,
	label string,
	onPCM func([]byte),
) (*RemoteTrack, error) {
	if track == nil {
		return nil, errors.New("pcm: track is nil")
	}
	if track.Codec().MimeType != webrtc.MimeTypeOpus {
		return nil, fmt.Errorf("pcm: unsupported codec %s", track.Codec().MimeType)
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}

	ctx, cancel := context.WithCancel(ctx)
	sink := NewSink(label, onPCM)
	rt := &RemoteTrack{cancel: cancel, sink: sink}

	go runDecodeLoop(ctx, track, sampleRate, sink)
	log.Printf("pcm: started %s sample_rate=%d", label, sampleRate)
	return rt, nil
}

func runDecodeLoop(
	ctx context.Context,
	track *webrtc.TrackRemote,
	targetRate int,
	sink *Sink,
) {
	decoder, err := gopus.NewDecoder(gopus.DefaultDecoderConfig(opusSampleRate, 1))
	if err != nil {
		log.Printf("pcm: decoder init: %v", err)
		return
	}

	builder := samplebuilder.New(200, &codecs.OpusPacket{}, opusSampleRate)
	pcmBuf := make([]int16, 5760)

	for {
		if ctx.Err() != nil {
			return
		}

		pkt, _, err := track.ReadRTP()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("pcm: rtp read: %v", err)
			}
			return
		}

		builder.Push(pkt)
		for sample := builder.Pop(); sample != nil; sample = builder.Pop() {
			if ctx.Err() != nil {
				return
			}
			if err := decodeSample(decoder, pcmBuf, sample.Data, targetRate, sink); err != nil {
				continue
			}
		}
	}
}

func decodeSample(
	decoder *gopus.Decoder,
	pcmBuf []int16,
	opusFrame []byte,
	targetRate int,
	sink *Sink,
) error {
	n, err := decoder.DecodeInt16(opusFrame, pcmBuf)
	if err != nil || n == 0 {
		return err
	}

	pcm48 := pcmBuf[:n]
	pcmOut := pcm48
	if targetRate != opusSampleRate {
		pcmOut = Resample(targetRate, opusSampleRate, pcm48)
	}
	return sink.WriteSample(pcmOut)
}
