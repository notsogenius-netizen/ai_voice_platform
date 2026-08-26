package pcm

import "testing"

func TestSampleToLinear16(t *testing.T) {
	got := SampleToLinear16([]int16{0, 1, -1, 32767})
	want := []byte{
		0x00, 0x00,
		0x01, 0x00,
		0xFF, 0xFF,
		0xFF, 0x7F,
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte[%d] = %#x, want %#x", i, got[i], want[i])
		}
	}
}

func TestSampleToLinear16Empty(t *testing.T) {
	if got := SampleToLinear16(nil); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
}

func TestSinkWriteSampleInvokesHandler(t *testing.T) {
	var chunks [][]byte
	sink := NewSink("test", func(b []byte) {
		buf := make([]byte, len(b))
		copy(buf, b)
		chunks = append(chunks, buf)
	})

	if err := sink.WriteSample([]int16{42, -42}); err != nil {
		t.Fatalf("WriteSample: %v", err)
	}
	if len(chunks) != 1 || len(chunks[0]) != 4 {
		t.Fatalf("chunks = %#v", chunks)
	}
}

func TestTrackSetRemove(t *testing.T) {
	set := NewTrackSet()
	sink := NewSink("test", nil)
	track := &RemoteTrack{sink: sink}

	set.Add("TR_test", track)
	set.Remove("TR_test")

	if err := sink.WriteSample([]int16{1}); err != nil {
		t.Fatalf("WriteSample after close: %v", err)
	}
}

func TestTrackSetCloseAll(t *testing.T) {
	set := NewTrackSet()
	sink := NewSink("test", nil)
	set.Add("TR_a", &RemoteTrack{sink: sink})

	set.CloseAll()
	set.Remove("TR_a")
}

func TestStartRemoteTrackRejectsNilTrack(t *testing.T) {
	if _, err := StartRemoteTrack(t.Context(), nil, 16000, "bad", nil); err == nil {
		t.Fatal("expected error for nil track")
	}
}

func TestResampleToTargetRate(t *testing.T) {
	in := []int16{1000, 2000, 3000, 4000, 5000, 6000}
	out := Resample(16000, 48000, in)
	if len(out) == 0 {
		t.Fatal("expected resampled output")
	}
	if len(out) >= len(in) {
		t.Fatalf("downsampled len = %d, want fewer than %d", len(out), len(in))
	}
}

func TestResampleSameRateCopies(t *testing.T) {
	in := []int16{1, 2, 3}
	out := Resample(48000, 48000, in)
	if len(out) != len(in) {
		t.Fatalf("len = %d, want %d", len(out), len(in))
	}
}
