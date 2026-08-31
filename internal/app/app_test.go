package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/65658dsf/SimpleTools/internal/tools"
)

func waitForTerminalJob(t *testing.T, a *App, id string) *JobStatus {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		status, err := a.GetJob(id)
		if err != nil {
			t.Fatal(err)
		}
		if status.State != "queued" && status.State != "running" {
			return status
		}
		if time.Now().After(deadline) {
			t.Fatal("job timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestStartJobConvertsPNGToJPEG(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "input.png")
	out := filepath.Join(d, "out")
	f, err := os.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	if err = png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if err = os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	a := New()
	id, err := a.StartJob(tools.JobRequest{Inputs: []string{in}, OutputDirectory: out, Format: "jpeg", Quality: 80})
	if err != nil {
		t.Fatal(err)
	}
	s := waitForTerminalJob(t, a, id)
	if s.State != "completed" {
		t.Fatalf("unexpected state %#v", s)
	}
	if s.Completed != 1 || len(s.Outputs) != 1 {
		t.Fatalf("bad status %#v", s)
	}
	if _, err = os.Stat(s.Outputs[0]); err != nil {
		t.Fatal(err)
	}
}

func TestCompressionJobReportsActualByteCounts(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "input.png")
	out := filepath.Join(d, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, 96, 64))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 2), G: uint8(y * 3), B: uint8(x + y), A: 255})
		}
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	sourceInfo, err := os.Stat(in)
	if err != nil {
		t.Fatal(err)
	}

	itemEvents := make(chan JobItem, 8)
	a := New()
	a.setEventSink(func(name string, payload any) {
		if name == "job:item" {
			itemEvents <- payload.(JobItem)
		}
	})
	id, err := a.StartJob(tools.JobRequest{
		Tool:            tools.ToolCompress,
		Inputs:          []string{in},
		OutputDirectory: out,
		Quality:         70,
	})
	if err != nil {
		t.Fatal(err)
	}
	status := waitForTerminalJob(t, a, id)
	if status.State != "completed" || len(status.Items) != 1 || len(status.Outputs) != 1 {
		t.Fatalf("unexpected status %#v", status)
	}
	outputInfo, err := os.Stat(status.Outputs[0])
	if err != nil {
		t.Fatal(err)
	}
	item := status.Items[0]
	if item.OriginalBytes != sourceInfo.Size() || item.CompressedBytes != outputInfo.Size() {
		t.Fatalf("unexpected job bytes: original=%d want=%d compressed=%d want=%d", item.OriginalBytes, sourceInfo.Size(), item.CompressedBytes, outputInfo.Size())
	}

	deadline := time.After(time.Second)
	for {
		select {
		case event := <-itemEvents:
			if event.State == "completed" {
				if event.OriginalBytes != sourceInfo.Size() || event.CompressedBytes != outputInfo.Size() {
					t.Fatalf("unexpected event bytes: %#v", event)
				}
				return
			}
		case <-deadline:
			t.Fatal("completed item event was not emitted")
		}
	}
}

func TestJobItemByteFieldsUseStableJSONNames(t *testing.T) {
	encoded, err := json.Marshal(JobItem{ID: "item-1", Path: "source.png", Name: "source.png", State: "completed", OriginalBytes: 1234, CompressedBytes: 567})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(encoded, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["originalBytes"] != float64(1234) || payload["compressedBytes"] != float64(567) {
		t.Fatalf("unexpected byte fields in JSON: %s", encoded)
	}
	if _, ok := payload["OriginalBytes"]; ok {
		t.Fatalf("Go field name leaked into JSON: %s", encoded)
	}
	zeroEncoded, err := json.Marshal(JobItem{ID: "item-2", State: "queued"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(zeroEncoded), "originalBytes") || strings.Contains(string(zeroEncoded), "compressedBytes") {
		t.Fatalf("zero byte fields should be omitted: %s", zeroEncoded)
	}
}

func TestPruneFinishedJobsHonorsTTLAndKeepsActiveJobs(t *testing.T) {
	a := New()
	a.jobRetention = time.Minute
	a.maxFinishedJobs = 50
	finishedAt := time.Now().Add(-2 * time.Minute)
	finished := &job{JobStatus: JobStatus{ID: "finished", State: "completed", FinishedAt: &finishedAt}}
	activeFinishedAt := finishedAt
	active := &job{JobStatus: JobStatus{ID: "active", State: "running", FinishedAt: &activeFinishedAt}}
	a.jobs[finished.ID] = finished
	a.jobs[active.ID] = active

	a.pruneFinishedJobs(time.Now())
	if _, err := a.GetJob(finished.ID); err == nil {
		t.Fatal("expired finished job was retained")
	}
	if _, err := a.GetJob(active.ID); err != nil {
		t.Fatalf("active job was pruned: %v", err)
	}
}

func TestPruneFinishedJobsKeepsNewestFinishedJobsWithinLimit(t *testing.T) {
	a := New()
	a.jobRetention = time.Hour
	a.maxFinishedJobs = 2
	now := time.Now()
	for i := 0; i < 3; i++ {
		finishedAt := now.Add(time.Duration(i) * time.Second)
		id := fmt.Sprintf("finished-%d", i)
		a.jobs[id] = &job{JobStatus: JobStatus{ID: id, State: "completed", FinishedAt: &finishedAt}}
	}
	active := &job{JobStatus: JobStatus{ID: "active", State: "running"}}
	a.jobs[active.ID] = active

	a.pruneFinishedJobs(now.Add(3 * time.Second))
	if _, err := a.GetJob("finished-0"); err == nil {
		t.Fatal("oldest finished job was retained beyond the configured limit")
	}
	for _, id := range []string{"finished-1", "finished-2", active.ID} {
		if _, err := a.GetJob(id); err != nil {
			t.Fatalf("job %q was unexpectedly pruned: %v", id, err)
		}
	}
}

func TestCompletedJobIsRemovedAfterRetention(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "input.png")
	out := filepath.Join(d, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	a := New()
	a.jobRetention = 20 * time.Millisecond
	id, err := a.StartJob(tools.JobRequest{Inputs: []string{in}, OutputDirectory: out, Format: "jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	if status := waitForTerminalJob(t, a, id); status.State != "completed" {
		t.Fatalf("unexpected terminal state %#v", status)
	}

	deadline := time.Now().Add(time.Second)
	for {
		if _, err := a.GetJob(id); err != nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("completed job was not removed after its retention period")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestJobProgressEventsUseLightweightPayloads(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "input.png")
	out := filepath.Join(d, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	progressEvents := make(chan JobProgress, 32)
	a := New()
	a.setEventSink(func(name string, payload any) {
		if name != "job:progress" {
			return
		}
		if progress, ok := payload.(JobProgress); ok {
			progressEvents <- progress
		}
	})
	id, err := a.StartJob(tools.JobRequest{Inputs: []string{in}, OutputDirectory: out, Format: "jpeg"})
	if err != nil {
		t.Fatal(err)
	}
	if status := waitForTerminalJob(t, a, id); status.State != "completed" {
		t.Fatalf("unexpected terminal status %#v", status)
	}
	foundItem := false
	for {
		select {
		case progress := <-progressEvents:
			if progress.Item != nil {
				foundItem = true
				if progress.Item.Path != in || progress.Total != 1 {
					t.Fatalf("unexpected lightweight progress %#v", progress)
				}
			}
		default:
			if !foundItem {
				t.Fatal("no changed item was included in progress events")
			}
			return
		}
	}
}

func TestStartJobUsesDefaultOutputDirectoryWhenUnset(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "input.png")
	f, err := os.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(d, "default-output")
	revealed := make(chan []string, 1)
	a := New()
	a.defaultOutputDir = func() (string, error) { return out, nil }
	a.revealOutputs = func(paths []string) error {
		revealed <- append([]string(nil), paths...)
		return nil
	}
	id, err := a.StartJob(tools.JobRequest{Inputs: []string{in}, Format: "jpeg", Quality: 80})
	if err != nil {
		t.Fatal(err)
	}
	status := waitForTerminalJob(t, a, id)
	if status.State != "completed" || len(status.Outputs) != 1 {
		t.Fatalf("unexpected status %#v", status)
	}
	expectedOut, err := filepath.EvalSymlinks(out)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(filepath.Dir(status.Outputs[0])) != filepath.Clean(expectedOut) {
		t.Fatalf("expected output below default directory %q, got %q", expectedOut, status.Outputs[0])
	}
	select {
	case paths := <-revealed:
		if len(paths) != 1 || paths[0] != status.Outputs[0] {
			t.Fatalf("unexpected revealed paths %#v", paths)
		}
	case <-time.After(time.Second):
		t.Fatal("output reveal was not requested")
	}
}

func TestExpandJobInputsRejectsRelativeDirectoryEscape(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "input.png")
	if err := os.WriteFile(in, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := expandJobInputs([]string{in}, "image", false, map[string]string{in: "..\\outside"})
	if err == nil {
		t.Fatal("expected relative directory traversal to be rejected")
	}
}

func TestExpandJobInputsRejectsSymlinkFileOutsideFolder(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	input := filepath.Join(outside, "outside.png")
	f, err := os.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(root, "linked.png")
	if err := os.Symlink(input, linked); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := expandJobInputs([]string{root}, "image", false); err == nil {
		t.Fatal("expected a symlink escaping the selected folder to be rejected")
	}
}

func TestStartJobKeepsExistingOutputAndReportsPartialFailure(t *testing.T) {
	d := t.TempDir()
	out := filepath.Join(d, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	good := filepath.Join(d, "good.png")
	f, err := os.Create(good)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(d, "bad.png")
	if err := os.WriteFile(bad, []byte("not a png"), 0o644); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(out, "good.jpg")
	if err := os.WriteFile(existing, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := New()
	id, err := a.StartJob(tools.JobRequest{Inputs: []string{good, bad}, OutputDirectory: out, Format: "jpg"})
	if err != nil {
		t.Fatal(err)
	}
	status := waitForTerminalJob(t, a, id)
	if status.State != "completed_with_errors" || status.Completed != 1 || status.Failed != 1 {
		t.Fatalf("unexpected status %#v", status)
	}
	if got, err := os.ReadFile(existing); err != nil || string(got) != "keep me" {
		t.Fatalf("existing output changed: %q, %v", got, err)
	}
	if len(status.Outputs) != 1 || !strings.HasSuffix(status.Outputs[0], "good-1.jpg") {
		t.Fatalf("expected collision-safe output, got %#v", status.Outputs)
	}
}

func TestWatermarkJobPreservesFormatAndUsesCollisionSafeName(t *testing.T) {
	d := t.TempDir()
	out := filepath.Join(d, "out")
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	in := filepath.Join(d, "photo.png")
	source := image.NewNRGBA(image.Rect(0, 0, 360, 220))
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 25, G: 45, B: 65, A: 255})
		}
	}
	f, err := os.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, source); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(out, "photo-watermarked.png")
	if err := os.WriteFile(existing, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := New()
	id, err := a.StartJob(tools.JobRequest{
		Tool:            tools.ToolWatermark,
		Inputs:          []string{in},
		OutputDirectory: out,
		Watermark: &tools.WatermarkOptions{
			Text: "SimpleTools 水印", FontFamily: "noto-sans-sc", FontSize: 42,
			Color: "#ffffff", Opacity: 0.8, Position: "bottom-right", Margin: 12, Shadow: true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	status := waitForTerminalJob(t, a, id)
	if status.State != "completed" || len(status.Outputs) != 1 {
		t.Fatalf("unexpected status %#v", status)
	}
	if got := filepath.Base(status.Outputs[0]); got != "photo-watermarked-1.png" {
		t.Fatalf("output name = %q", got)
	}
	if kept, err := os.ReadFile(existing); err != nil || string(kept) != "keep me" {
		t.Fatalf("existing output changed: %q, %v", kept, err)
	}
	generated, err := os.Open(status.Outputs[0])
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(generated)
	_ = generated.Close()
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds() != source.Bounds() {
		t.Fatalf("output bounds = %v, want %v", decoded.Bounds(), source.Bounds())
	}
	if imagesEqual(decoded, source) {
		t.Fatal("watermarked output is identical to its source")
	}
}

func TestWatermarkJobRequiresOptions(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "input.png")
	f, err := os.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	_ = f.Close()
	if _, err := New().StartJob(tools.JobRequest{Tool: tools.ToolWatermark, Inputs: []string{in}, OutputDirectory: d}); err == nil || !strings.Contains(err.Error(), "watermark options") {
		t.Fatalf("expected missing watermark options error, got %v", err)
	}
}

func TestPreviewWatermarkReturnsBoundedComparison(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "preview.png")
	source := image.NewNRGBA(image.Rect(0, 0, 900, 540))
	for y := 0; y < source.Bounds().Dy(); y++ {
		for x := 0; x < source.Bounds().Dx(); x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: uint8(x + y), A: 255})
		}
	}
	f, err := os.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, source); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	preview, err := New().PreviewWatermark(in, tools.WatermarkOptions{
		Text: "预览", FontFamily: "noto-sans-sc", FontSize: 96, Color: "#ffffff",
		Opacity: 0.8, Position: "center", Rotation: -20, Shadow: true,
	}, 320)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Path != in || preview.Width != 900 || preview.Height != 540 || preview.Truncated {
		t.Fatalf("unexpected preview metadata %#v", preview)
	}
	before := decodeImageDataURL(t, preview.BeforeDataURL)
	after := decodeImageDataURL(t, preview.AfterDataURL)
	if before.Bounds() != after.Bounds() || before.Bounds().Dx() > 320 || before.Bounds().Dy() > 320 {
		t.Fatalf("preview bounds before=%v after=%v", before.Bounds(), after.Bounds())
	}
	if imagesEqual(before, after) {
		t.Fatal("before and after previews are identical")
	}
	const maxDataURLLength = 1024*1024 + 64
	if len(preview.BeforeDataURL) > maxDataURLLength || len(preview.AfterDataURL) > maxDataURLLength {
		t.Fatalf("preview data URLs are not bounded: before=%d after=%d", len(preview.BeforeDataURL), len(preview.AfterDataURL))
	}
}

func TestPreviewWatermarkPreservesTransparency(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "transparent.png")
	source := image.NewNRGBA(image.Rect(0, 0, 320, 180))
	f, err := os.Create(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, source); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	preview, err := New().PreviewWatermark(in, tools.WatermarkOptions{
		Text: "alpha", FontFamily: "noto-sans-sc", FontSize: 72, Color: "#ffffff",
		Opacity: 0.5, Position: "center",
	}, 320)
	if err != nil {
		t.Fatal(err)
	}
	before := decodeImageDataURL(t, preview.BeforeDataURL)
	after := decodeImageDataURL(t, preview.AfterDataURL)
	if before.Bounds() != source.Bounds() || after.Bounds() != source.Bounds() {
		t.Fatalf("preview bounds before=%v after=%v", before.Bounds(), after.Bounds())
	}

	watermarkAlpha := false
	for y := before.Bounds().Min.Y; y < before.Bounds().Max.Y; y++ {
		for x := before.Bounds().Min.X; x < before.Bounds().Max.X; x++ {
			_, _, _, beforeAlpha := before.At(x, y).RGBA()
			if beforeAlpha != 0 {
				t.Fatalf("before preview lost transparency at (%d, %d): alpha=%d", x, y, beforeAlpha)
			}
			_, _, _, afterAlpha := after.At(x, y).RGBA()
			if afterAlpha > 0 && afterAlpha < 0xffff {
				watermarkAlpha = true
			}
		}
	}
	if !watermarkAlpha {
		t.Fatal("after preview contains no semi-transparent watermark pixels")
	}
}

func TestWatermarkPreviewPNGShrinksToPayloadLimit(t *testing.T) {
	before := image.NewNRGBA(image.Rect(0, 0, 1024, 1024))
	after := image.NewNRGBA(before.Bounds())
	state := uint32(0x12345678)
	for y := 0; y < before.Bounds().Dy(); y++ {
		for x := 0; x < before.Bounds().Dx(); x++ {
			state ^= state << 13
			state ^= state >> 17
			state ^= state << 5
			before.SetNRGBA(x, y, color.NRGBA{R: uint8(state), G: uint8(state >> 8), B: uint8(state >> 16), A: uint8(state >> 24)})
			after.SetNRGBA(x, y, color.NRGBA{R: uint8(state >> 24), G: uint8(state >> 16), B: uint8(state >> 8), A: uint8(state)})
		}
	}

	beforeURL, afterURL, truncated, err := encodeWatermarkPreviewPair(before, after, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Fatal("bounded PNG preview was unexpectedly truncated")
	}
	const maxDataURLLength = 1024*1024 + 64
	if len(beforeURL) > maxDataURLLength || len(afterURL) > maxDataURLLength {
		t.Fatalf("preview data URLs exceed the payload limit: before=%d after=%d", len(beforeURL), len(afterURL))
	}
	beforePreview := decodeImageDataURL(t, beforeURL)
	afterPreview := decodeImageDataURL(t, afterURL)
	if beforePreview.Bounds() != afterPreview.Bounds() {
		t.Fatalf("preview bounds differ: before=%v after=%v", beforePreview.Bounds(), afterPreview.Bounds())
	}
	if beforePreview.Bounds().Dx() >= before.Bounds().Dx() || beforePreview.Bounds().Dy() >= before.Bounds().Dy() {
		t.Fatalf("high-entropy PNG preview was not reduced: %v", beforePreview.Bounds())
	}
}

func TestPreviewQRCodeReturnsBoundedPNGWithRequestedColors(t *testing.T) {
	preview, err := New().PreviewQRCode(tools.QRCodeOptions{
		Text:            "SimpleTools 二维码预览",
		Size:            512,
		ErrorCorrection: "quartile",
		Foreground:      "#102A43",
		Background:      "#F5F7FA",
	}, 240)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Size != 240 {
		t.Fatalf("preview size = %d, want 240", preview.Size)
	}
	if !strings.HasPrefix(preview.DataURL, "data:image/png;base64,") {
		t.Fatalf("unexpected data URL prefix: %.32q", preview.DataURL)
	}
	const maxDataURLLength = 1024*1024 + 64
	if len(preview.DataURL) > maxDataURLLength {
		t.Fatalf("preview data URL is not bounded: %d", len(preview.DataURL))
	}

	decoded := decodeImageDataURL(t, preview.DataURL)
	wantForeground := color.RGBA{R: 0x10, G: 0x2a, B: 0x43, A: 0xff}
	wantBackground := color.RGBA{R: 0xf5, G: 0xf7, B: 0xfa, A: 0xff}
	foundForeground, foundBackground := false, false
	for y := decoded.Bounds().Min.Y; y < decoded.Bounds().Max.Y && (!foundForeground || !foundBackground); y++ {
		for x := decoded.Bounds().Min.X; x < decoded.Bounds().Max.X; x++ {
			pixel := color.RGBAModel.Convert(decoded.At(x, y)).(color.RGBA)
			foundForeground = foundForeground || pixel == wantForeground
			foundBackground = foundBackground || pixel == wantBackground
		}
	}
	if !foundForeground || !foundBackground {
		t.Fatalf("preview colors foreground=%v background=%v", foundForeground, foundBackground)
	}
}

func TestSaveQRCodeUsesDefaultDirectorySafeNameAndCollisionSuffix(t *testing.T) {
	temp := t.TempDir()
	output := filepath.Join(temp, "generated")
	a := New()
	a.defaultOutputDir = func() (string, error) { return output, nil }
	options := tools.QRCodeOptions{
		Text:            "https://example.test/simple-tools",
		Size:            256,
		ErrorCorrection: "medium",
		Foreground:      "#000000",
		Background:      "#FFFFFF",
	}

	first, err := a.SaveQRCode(options, "", `..\reports\Quarter:Report?.PNG`)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(output, "Quarter-Report.png"); first != want {
		t.Fatalf("first output = %q, want %q", first, want)
	}
	second, err := a.SaveQRCode(options, "", `..\reports\Quarter:Report?.PNG`)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(output, "Quarter-Report-1.png"); second != want {
		t.Fatalf("collision output = %q, want %q", second, want)
	}
	defaultName, err := a.SaveQRCode(options, "", "   ")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(output, "qrcode.png"); defaultName != want {
		t.Fatalf("default output = %q, want %q", defaultName, want)
	}

	for _, generated := range []string{first, second, defaultName} {
		file, err := os.Open(generated)
		if err != nil {
			t.Fatal(err)
		}
		decoded, decodeErr := png.Decode(file)
		closeErr := file.Close()
		if decodeErr != nil {
			t.Fatalf("decode %q: %v", generated, decodeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if got := decoded.Bounds().Size(); got.X != options.Size || got.Y != options.Size {
			t.Fatalf("saved QR size = %v, want %dx%d", got, options.Size, options.Size)
		}
	}
}

func TestSanitizeQRCodeOutputBaseHandlesUnsafeAndReservedNames(t *testing.T) {
	tests := map[string]string{
		"":                        "qrcode",
		"../../.png":              "qrcode",
		`folder\safe-name.png`:    "safe-name",
		"  quarterly:report?.PNG": "quarterly-report",
		"CON.png":                 "_CON",
		"LPT9.notes":              "_LPT9.notes",
	}
	for input, want := range tests {
		if got := sanitizeQRCodeOutputBase(input); got != want {
			t.Errorf("sanitizeQRCodeOutputBase(%q) = %q, want %q", input, got, want)
		}
	}
}

func decodeImageDataURL(t *testing.T, value string) image.Image {
	t.Helper()
	header, payload, ok := strings.Cut(value, ",")
	if !ok || !strings.HasPrefix(header, "data:image/") || !strings.HasSuffix(header, ";base64") {
		t.Fatalf("unexpected preview data URL header %q", header)
	}
	data, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return decoded
}

func imagesEqual(first, second image.Image) bool {
	if first.Bounds() != second.Bounds() {
		return false
	}
	for y := first.Bounds().Min.Y; y < first.Bounds().Max.Y; y++ {
		for x := first.Bounds().Min.X; x < first.Bounds().Max.X; x++ {
			if first.At(x, y) != second.At(x, y) {
				return false
			}
		}
	}
	return true
}

type fakePDFRenderer struct {
	pages   []tools.PDFPage
	started chan struct{}
	block   bool
}

func (r *fakePDFRenderer) Render(ctx context.Context, _ string, _ tools.PDFOptions, onPage func(tools.PDFPage) error) error {
	if r.started != nil {
		close(r.started)
	}
	if r.block {
		<-ctx.Done()
		return ctx.Err()
	}
	for _, page := range r.pages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := onPage(page); err != nil {
			return err
		}
	}
	return nil
}

func TestPDFJobWritesSelectedPagesAndReportsProgress(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "document.pdf")
	out := filepath.Join(d, "out")
	if err := os.WriteFile(in, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	renderer := &fakePDFRenderer{pages: []tools.PDFPage{
		{Number: 1, Index: 0, Total: 2, PNG: []byte("page-one")},
		{Number: 3, Index: 1, Total: 2, PNG: []byte("page-three")},
	}}
	a := New()
	a.pdfRender = renderer
	id, err := a.StartJob(tools.JobRequest{Tool: tools.ToolPDF, Inputs: []string{in}, OutputDirectory: out, PageRange: "1,3"})
	if err != nil {
		t.Fatal(err)
	}
	status := waitForTerminalJob(t, a, id)
	if status.State != "completed" || len(status.Outputs) != 2 {
		t.Fatalf("unexpected status %#v", status)
	}
	if filepath.Base(status.Outputs[0]) != "page-001.png" || filepath.Base(status.Outputs[1]) != "page-003.png" {
		t.Fatalf("unexpected page outputs %#v", status.Outputs)
	}
	for _, path := range status.Outputs {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPDFJobCancellationCleansPartialDirectory(t *testing.T) {
	d := t.TempDir()
	in := filepath.Join(d, "document.pdf")
	out := filepath.Join(d, "out")
	if err := os.WriteFile(in, []byte("fixture"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(out, 0o755); err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	a := New()
	a.pdfRender = &fakePDFRenderer{started: started, block: true}
	id, err := a.StartJob(tools.JobRequest{Tool: tools.ToolPDF, Inputs: []string{in}, OutputDirectory: out})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("renderer did not start")
	}
	if err := a.CancelJob(id); err != nil {
		t.Fatal(err)
	}
	status := waitForTerminalJob(t, a, id)
	if status.State != "cancelled" || status.Failed != 0 {
		t.Fatalf("unexpected cancellation status %#v", status)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("partial PDF output remains: %#v", entries)
	}
}
