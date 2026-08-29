package app

import (
	"context"
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
