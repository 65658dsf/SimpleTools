package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/65658dsf/SimpleTools/internal/platform"
	"github.com/65658dsf/SimpleTools/internal/tools"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type EventSink func(name string, payload any)

type App struct {
	mu         sync.RWMutex
	jobs       map[string]*job
	sink       EventSink
	ctx        context.Context
	updater    platform.UpdateSource
	pdfRender  tools.PDFRenderer
	imageSlots chan struct{}
	pdfSlots   chan struct{}
	version    string
	lastUpdate *UpdateInfo
}

type InputFile struct {
	Path         string `json:"path"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	Kind         string `json:"kind"`
	RelativePath string `json:"relativePath,omitempty"`
}

type PreviewOptions struct {
	MaxDimension int `json:"maxDimension,omitempty"`
}

type Preview struct {
	Path      string `json:"path"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Format    string `json:"format"`
	Size      int64  `json:"size"`
	DataURL   string `json:"dataUrl,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

type JobItem struct {
	ID       string   `json:"id"`
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	State    string   `json:"state"`
	Progress float64  `json:"progress"`
	Output   string   `json:"output,omitempty"`
	Outputs  []string `json:"outputs,omitempty"`
	Error    string   `json:"error,omitempty"`
	Warning  string   `json:"warning,omitempty"`
}

type JobStatus struct {
	ID         string     `json:"id"`
	State      string     `json:"state"`
	Total      int        `json:"total"`
	Completed  int        `json:"completed"`
	Failed     int        `json:"failed"`
	Progress   float64    `json:"progress"`
	Current    string     `json:"current,omitempty"`
	Error      string     `json:"error,omitempty"`
	Outputs    []string   `json:"outputs,omitempty"`
	Items      []JobItem  `json:"items"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

type JobRequest = tools.JobRequest
type UpdateInfo = platform.UpdateInfo

type jobInput struct {
	Path   string
	RelDir string
	Name   string
}

type jobResult struct {
	Index   int
	State   string
	Outputs []string
	Warning string
	Err     error
}

type job struct {
	JobStatus
	cancel context.CancelFunc
	inputs []jobInput
}

type outputAllocator struct {
	mu          sync.Mutex
	reserved    map[string]struct{}
	directories map[string]struct{}
}

func newOutputAllocator() *outputAllocator {
	return &outputAllocator{reserved: map[string]struct{}{}, directories: map[string]struct{}{}}
}

func (a *outputAllocator) next(dir, base, ext string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := 0; ; i++ {
		candidateBase := base
		if i > 0 {
			candidateBase = fmt.Sprintf("%s-%d", base, i)
		}
		candidate := filepath.Join(dir, candidateBase+ext)
		if _, exists := a.reserved[candidate]; exists {
			continue
		}
		if _, err := os.Lstat(candidate); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect output path %q: %w", candidate, err)
		}
		a.reserved[candidate] = struct{}{}
		return candidate, nil
	}
}

func (a *outputAllocator) directory(parent, base string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := 0; ; i++ {
		name := base
		if i > 0 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		candidate := filepath.Join(parent, name)
		if _, reserved := a.directories[candidate]; reserved {
			continue
		}
		if _, err := os.Lstat(candidate); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect output directory %q: %w", candidate, err)
		}
		a.directories[candidate] = struct{}{}
		return candidate, nil
	}
}

func New(versions ...string) *App {
	currentVersion := "0.1.0"
	if len(versions) > 0 && strings.TrimSpace(versions[0]) != "" {
		currentVersion = strings.TrimSpace(versions[0])
	}
	owner, repository := os.Getenv("SIMPLETEOOLS_UPDATE_OWNER"), os.Getenv("SIMPLETEOOLS_UPDATE_REPOSITORY")
	if owner == "" {
		owner = "65658dsf"
	}
	if repository == "" {
		repository = "SimpleTools"
	}
	updater := platform.NewGitHubUpdater(owner, repository)
	publicKey := os.Getenv("SIMPLETEOOLS_UPDATE_PUBLIC_KEY")
	if len(versions) > 1 && strings.TrimSpace(versions[1]) != "" {
		publicKey = strings.TrimSpace(versions[1])
	}
	_ = updater.SetPublicKey(publicKey)
	return &App{
		jobs:       map[string]*job{},
		updater:    updater,
		pdfRender:  tools.DefaultPDFRenderer(),
		imageSlots: make(chan struct{}, 4),
		pdfSlots:   make(chan struct{}, 2),
		version:    currentVersion,
	}
}

// Startup connects the app to the Wails runtime. Tests can continue to inject
// an EventSink directly without creating a native window.
func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.sink = func(name string, payload any) { wailsruntime.EventsEmit(ctx, name, payload) }
	a.mu.Unlock()
	go func() {
		info, err := a.CheckForUpdate()
		if err == nil && info != nil && info.Available {
			a.emit("update:available", info)
		}
	}()
}

func (a *App) Shutdown(context.Context) {
	a.mu.RLock()
	jobs := make([]*job, 0, len(a.jobs))
	for _, j := range a.jobs {
		jobs = append(jobs, j)
	}
	a.mu.RUnlock()
	for _, j := range jobs {
		j.cancel()
	}
}

func (a *App) setEventSink(s EventSink) { a.mu.Lock(); a.sink = s; a.mu.Unlock() }

func (a *App) emit(name string, payload any) {
	a.mu.RLock()
	sink := a.sink
	a.mu.RUnlock()
	if sink != nil {
		sink(name, payload)
	}
}

func (a *App) dialogContext() (context.Context, error) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx == nil {
		return nil, errors.New("application is not started")
	}
	return ctx, nil
}

func (a *App) OpenInputFiles() ([]InputFile, error) {
	ctx, err := a.dialogContext()
	if err != nil {
		return nil, err
	}
	paths, err := wailsruntime.OpenMultipleFilesDialog(ctx, wailsruntime.OpenDialogOptions{
		Title:   "Select files",
		Filters: []wailsruntime.FileFilter{{DisplayName: "Supported files", Pattern: "*.png;*.jpg;*.jpeg;*.webp;*.avif;*.pdf"}},
	})
	if err != nil {
		return nil, err
	}
	return a.OpenInputFilesFromPaths(paths)
}

func (a *App) OpenInputFolder() ([]InputFile, error) {
	ctx, err := a.dialogContext()
	if err != nil {
		return nil, err
	}
	path, err := wailsruntime.OpenDirectoryDialog(ctx, wailsruntime.OpenDialogOptions{Title: "Select input folder"})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(path) == "" {
		return []InputFile{}, nil
	}
	return a.openInputPaths([]string{path}, "", true)
}

func (a *App) OpenInputFilesFromPaths(paths []string) ([]InputFile, error) {
	// OS drag-and-drop can include directories. Treat those as folder inputs so
	// nested supported files are discoverable just like OpenInputFolder.
	return a.openInputPaths(paths, "", true)
}

func (a *App) openInputPaths(paths []string, kind string, recursive bool) ([]InputFile, error) {
	var out []InputFile
	for _, path := range paths {
		path = filepath.Clean(path)
		st, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if st.IsDir() {
			files, err := platform.CollectPaths(path, kind, recursive)
			if err != nil {
				return nil, err
			}
			for _, file := range files {
				out = append(out, inputMetadata(file, path))
			}
			continue
		}
		if kind == "pdf" && !platform.IsPDFPath(path) || kind == "image" && !platform.IsImagePath(path) {
			continue
		}
		if kind == "" && !platform.IsImagePath(path) && !platform.IsPDFPath(path) {
			continue
		}
		out = append(out, inputMetadata(path, ""))
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path) })
	return out, nil
}

func inputMetadata(path, root string) InputFile {
	if canonical, err := canonicalInputPath(path); err == nil {
		path = canonical
	}
	if root != "" {
		if canonicalRoot, err := canonicalInputPath(root); err == nil {
			root = canonicalRoot
		}
	}
	st, _ := os.Stat(path)
	relative := ""
	if root != "" {
		relative, _ = filepath.Rel(root, filepath.Dir(path))
		if relative == "." {
			relative = ""
		}
	}
	kind := "image"
	if platform.IsPDFPath(path) {
		kind = "pdf"
	}
	return InputFile{Path: path, Name: filepath.Base(path), Size: st.Size(), Kind: kind, RelativePath: relative}
}

func (a *App) ChooseOutputDirectory() (string, error) {
	ctx, err := a.dialogContext()
	if err != nil {
		return "", err
	}
	path, err := wailsruntime.OpenDirectoryDialog(ctx, wailsruntime.OpenDialogOptions{Title: "Select output folder", CanCreateDirectories: true})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errors.New("output directory is required")
	}
	return validateOutputDirectory(path)
}

func (a *App) OpenOutputDirectory(path string) error {
	path, err := validateOutputDirectory(path)
	if err != nil {
		return err
	}
	return platform.OpenDirectory(path)
}

func validateOutputDirectory(path string) (string, error) {
	path, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	st, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", errors.New("path is not a directory")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve output directory: %w", err)
	}
	return filepath.Abs(filepath.Clean(resolved))
}

func (a *App) PreviewImage(path string, options PreviewOptions) (*Preview, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	img, err := tools.Decode(f, ext)
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	preview := &Preview{Path: path, Width: bounds.Dx(), Height: bounds.Dy(), Format: ext, Size: st.Size()}
	maxDimension := options.MaxDimension
	if maxDimension <= 0 || maxDimension > 1024 {
		maxDimension = 512
	}
	thumb := thumbnail(img, maxDimension)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, flattenPreview(thumb), &jpeg.Options{Quality: 78}); err == nil && buf.Len() <= 768*1024 {
		preview.DataURL = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	} else {
		preview.Truncated = true
	}
	return preview, nil
}

func thumbnail(src image.Image, maxDimension int) image.Image {
	b := src.Bounds()
	if b.Dx() <= maxDimension && b.Dy() <= maxDimension {
		return src
	}
	scale := float64(maxDimension) / float64(max(b.Dx(), b.Dy()))
	dw, dh := max(1, int(float64(b.Dx())*scale)), max(1, int(float64(b.Dy())*scale))
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			sx := b.Min.X + x*b.Dx()/dw
			sy := b.Min.Y + y*b.Dy()/dh
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func flattenPreview(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, dst.Bounds(), image.NewUniform(image.White), image.Point{}, draw.Src)
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Over)
	return dst
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a *App) StartJob(req tools.JobRequest) (string, error) {
	format, err := req.Validate()
	if err != nil {
		return "", err
	}
	outputDir, err := validateOutputDirectory(req.OutputDirectory)
	if err != nil {
		return "", fmt.Errorf("output directory: %w", err)
	}
	req.OutputDirectory = outputDir
	kind := "image"
	if req.ToolKind() == tools.ToolPDF {
		kind = "pdf"
	}
	inputs, err := expandJobInputs(req.Inputs, kind, req.Recursive, req.InputRelativeDirs)
	if err != nil {
		return "", err
	}
	if len(inputs) == 0 {
		return "", fmt.Errorf("no supported %s files found", kind)
	}
	id := fmt.Sprintf("job-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	items := make([]JobItem, len(inputs))
	for i, input := range inputs {
		items[i] = JobItem{ID: fmt.Sprintf("%s-item-%d", id, i+1), Path: input.Path, Name: input.Name, State: "queued"}
	}
	j := &job{JobStatus: JobStatus{ID: id, State: "queued", Total: len(inputs), Items: items, StartedAt: time.Now()}, cancel: cancel, inputs: inputs}
	a.mu.Lock()
	a.jobs[id] = j
	a.mu.Unlock()
	go a.run(ctx, j, req, format)
	return id, nil
}

func expandJobInputs(paths []string, kind string, recursive bool, relativeDirs ...map[string]string) ([]jobInput, error) {
	seen := map[string]struct{}{}
	var out []jobInput
	var requestedRelativeDirs map[string]string
	if len(relativeDirs) > 0 {
		requestedRelativeDirs = relativeDirs[0]
	}
	for _, raw := range paths {
		path := filepath.Clean(raw)
		st, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if !st.IsDir() {
			if kind == "image" && !platform.IsImagePath(path) || kind == "pdf" && !platform.IsPDFPath(path) {
				return nil, fmt.Errorf("unsupported input type: %s", filepath.Ext(path))
			}
			canonical, err := canonicalInputPath(path)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[inputPathKey(canonical)]; !ok {
				seen[inputPathKey(canonical)] = struct{}{}
				relDir, relErr := requestedRelativeDir(requestedRelativeDirs, path, canonical)
				if relErr != nil {
					return nil, relErr
				}
				out = append(out, jobInput{Path: canonical, Name: filepath.Base(canonical), RelDir: relDir})
			}
			continue
		}
		files, err := platform.CollectPaths(path, kind, recursive)
		if err != nil {
			return nil, err
		}
		root, err := canonicalInputPath(path)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			canonical, err := canonicalInputPath(file)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[inputPathKey(canonical)]; ok {
				continue
			}
			seen[inputPathKey(canonical)] = struct{}{}
			canonicalDir := filepath.Dir(canonical)
			withinRoot, err := platform.IsWithin(root, canonicalDir)
			if err != nil {
				return nil, fmt.Errorf("validate input directory %q: %w", canonicalDir, err)
			}
			if !withinRoot {
				return nil, fmt.Errorf("input path %q escapes selected folder", canonical)
			}
			rel, err := filepath.Rel(root, canonicalDir)
			if err != nil {
				return nil, fmt.Errorf("calculate input relative directory: %w", err)
			}
			if rel == "." {
				rel = ""
			}
			out = append(out, jobInput{Path: canonical, Name: filepath.Base(canonical), RelDir: rel})
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Path) < strings.ToLower(out[j].Path) })
	return out, nil
}

func canonicalInputPath(path string) (string, error) {
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return filepath.Abs(filepath.Clean(resolved))
	}
	return abs, nil
}

func inputPathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func requestedRelativeDir(relativeDirs map[string]string, rawPath, canonicalPath string) (string, error) {
	if len(relativeDirs) == 0 {
		return "", nil
	}
	value, ok := relativeDirs[rawPath]
	if !ok {
		value, ok = relativeDirs[canonicalPath]
	}
	if !ok || strings.TrimSpace(value) == "" || value == "." {
		return "", nil
	}
	value = filepath.Clean(value)
	if filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("input relative directory escapes output root: %q", value)
	}
	return value, nil
}

func (a *App) run(ctx context.Context, j *job, req tools.JobRequest, format tools.Format) {
	a.update(j, func(s *JobStatus) { s.State = "running" })
	allocator := newOutputAllocator()
	workerCount := 4
	if req.ToolKind() == tools.ToolPDF {
		workerCount = 2
	}
	tasks := make(chan int)
	results := make(chan jobResult, len(j.inputs))
	var workers sync.WaitGroup
	for n := 0; n < workerCount; n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range tasks {
				result := a.processOne(ctx, j, j.inputs[index], req, format, allocator)
				results <- jobResult{Index: index, State: result.state, Outputs: result.outputs, Warning: result.warning, Err: result.err}
			}
		}()
	}
	go func() {
		defer close(tasks)
		for index := range j.inputs {
			select {
			case tasks <- index:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()
	for result := range results {
		a.applyResult(j, result)
	}
	if ctx.Err() != nil {
		a.update(j, func(s *JobStatus) {
			s.State = "cancelled"
			for i := range s.Items {
				if s.Items[i].State == "queued" || s.Items[i].State == "processing" {
					s.Items[i].State = "cancelled"
				}
			}
		})
	} else {
		a.update(j, func(s *JobStatus) {
			if s.Failed > 0 {
				s.State = "completed_with_errors"
			} else {
				s.State = "completed"
			}
		})
	}
	a.finish(j)
	status := a.snapshot(j)
	a.emit("job:completed", status)
	if status.Failed > 0 {
		a.emit("job:failed", status)
	}
}

type processed struct {
	state   string
	outputs []string
	warning string
	err     error
}

func (a *App) processOne(ctx context.Context, j *job, input jobInput, req tools.JobRequest, format tools.Format, allocator *outputAllocator) processed {
	if err := ctx.Err(); err != nil {
		return processed{state: "cancelled", err: err}
	}
	release, acquired := a.acquireToolSlot(ctx, req.ToolKind())
	if !acquired {
		return processed{state: "cancelled", err: ctx.Err()}
	}
	defer release()
	a.updateItem(j, input.Path, "processing", 0, "", "")
	if req.ToolKind() == tools.ToolPDF {
		return a.processPDF(ctx, j, input, req, allocator)
	}
	src, err := os.Open(input.Path)
	if err != nil {
		return processed{state: "failed", err: err}
	}
	defer src.Close()
	img, err := tools.Decode(src, filepath.Ext(input.Path))
	if err != nil {
		return processed{state: "failed", err: err}
	}
	outputFormat := format
	if req.ToolKind() == tools.ToolCompress {
		outputFormat, err = tools.FormatFromPath(input.Path)
		if err != nil {
			return processed{state: "failed", err: err}
		}
	}
	ext := "." + string(outputFormat)
	if outputFormat == tools.FormatJPEG {
		ext = ".jpg"
	}
	dir := filepath.Join(req.OutputDirectory, input.RelDir)
	base := tools.OutputBase(input.Name)
	out, allocErr := allocator.next(dir, base, ext)
	if allocErr != nil {
		return processed{state: "failed", err: allocErr}
	}
	result := processed{state: "completed", outputs: []string{out}}
	if req.PreserveMetadata {
		result.warning = "metadata preservation is not supported by the current encoders; metadata was removed"
	}
	var data []byte
	if req.ToolKind() == tools.ToolCompress {
		compressed, compressErr := tools.CompressImage(img, outputFormat, req.Quality, req.TargetBytes, req.Lossless)
		if compressErr != nil {
			return processed{state: "failed", err: compressErr}
		}
		data, result.warning = compressed.Data, joinWarnings(result.warning, compressed.Warning)
	} else {
		data, err = tools.EncodeBytes(img, outputFormat, tools.EncodeOptions{Quality: req.Quality, Lossless: req.Lossless})
		if err != nil {
			return processed{state: "failed", err: err}
		}
	}
	if err := ctx.Err(); err != nil {
		return processed{state: "cancelled", err: err}
	}
	if err := platform.AtomicWrite(dir, filepath.Base(out), func(w io.Writer) error { _, err := io.Copy(w, bytes.NewReader(data)); return err }); err != nil {
		return processed{state: "failed", err: err}
	}
	return result
}

// acquireToolSlot enforces the process-wide media concurrency limit even when
// callers start more than one job. A nil channel keeps manually constructed App
// values usable in tests and embedders while New supplies the production limits.
func (a *App) acquireToolSlot(ctx context.Context, tool tools.Tool) (func(), bool) {
	var slots chan struct{}
	if tool == tools.ToolPDF {
		slots = a.pdfSlots
	} else {
		slots = a.imageSlots
	}
	if slots == nil {
		return func() {}, true
	}
	select {
	case slots <- struct{}{}:
		return func() { <-slots }, true
	case <-ctx.Done():
		return func() {}, false
	}
}

func (a *App) processPDF(ctx context.Context, j *job, input jobInput, req tools.JobRequest, allocator *outputAllocator) processed {
	baseDir := filepath.Join(req.OutputDirectory, input.RelDir)
	pdfDir, allocErr := allocator.directory(baseDir, tools.OutputBase(input.Name)+"-png")
	if allocErr != nil {
		return processed{state: "failed", err: allocErr}
	}
	result := processed{state: "completed"}
	pageCount := 0
	err := a.pdfRender.Render(ctx, input.Path, tools.PDFOptions{DPI: req.DPI, PageRange: req.PageRange, MaxPixels: req.MaxPixels}, func(page tools.PDFPage) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		pageCount++
		name := fmt.Sprintf("page-%03d.png", page.Number)
		out, allocErr := allocator.next(pdfDir, strings.TrimSuffix(name, ".png"), ".png")
		if allocErr != nil {
			return allocErr
		}
		if err := platform.AtomicWrite(pdfDir, filepath.Base(out), func(w io.Writer) error { _, err := io.Copy(w, bytes.NewReader(page.PNG)); return err }); err != nil {
			return err
		}
		result.outputs = append(result.outputs, out)
		progress := 0.0
		if page.Total > 0 {
			progress = float64(page.Index+1) / float64(page.Total)
		} else if pageCount > 0 {
			// Keep compatibility with custom renderers that do not populate the
			// optional sequence fields.
			progress = 0.01 * float64(pageCount)
		}
		a.updateItem(j, input.Path, "processing", progress, out, "")
		return nil
	})
	if err != nil {
		// pdfDir is allocated uniquely for this item, so removing it cannot
		// affect a prior run. This prevents partial page sets after cancellation
		// or a renderer error.
		_ = os.RemoveAll(pdfDir)
		if ctx.Err() != nil {
			return processed{state: "cancelled", err: ctx.Err()}
		}
		return processed{state: "failed", err: err}
	}
	if pageCount == 0 {
		_ = os.RemoveAll(pdfDir)
		return processed{state: "failed", err: errors.New("PDF contains no selected pages")}
	}
	return result
}

func uniqueDirectory(parent, base string) string {
	for i := 0; ; i++ {
		name := base
		if i > 0 {
			name = fmt.Sprintf("%s-%d", base, i)
		}
		candidate := filepath.Join(parent, name)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func joinWarnings(first, second string) string {
	if first == "" {
		return second
	}
	if second == "" {
		return first
	}
	return first + "; " + second
}

func (a *App) applyResult(j *job, result jobResult) {
	a.mu.Lock()
	if result.Index < 0 || result.Index >= len(j.Items) {
		a.mu.Unlock()
		return
	}
	item := &j.Items[result.Index]
	item.State = result.State
	if result.State == "completed" || result.State == "failed" {
		item.Progress = 1
	}
	item.Outputs = append([]string(nil), result.Outputs...)
	if len(result.Outputs) > 0 {
		item.Output = result.Outputs[len(result.Outputs)-1]
		j.Outputs = append(j.Outputs, result.Outputs...)
	}
	if result.Warning != "" {
		item.Warning = result.Warning
	}
	if result.Err != nil && !errors.Is(result.Err, context.Canceled) {
		item.Error = result.Err.Error()
	}
	if result.State == "completed" {
		j.Completed++
	} else if result.State == "failed" {
		j.Failed++
		if result.Err != nil {
			j.Error = result.Err.Error()
		}
	}
	j.Progress = float64(j.Completed+j.Failed) / float64(max(1, j.Total))
	j.Current = ""
	itemSnapshot := *item
	snapshot := j.JobStatus
	snapshot.Items = cloneJobItems(j.Items)
	snapshot.Outputs = append([]string(nil), j.Outputs...)
	a.mu.Unlock()
	a.emit("job:item", itemSnapshot)
	a.emit("job:progress", snapshot)
}

func (a *App) updateItem(j *job, path, state string, progress float64, output, errText string) {
	a.mu.Lock()
	for i := range j.Items {
		if j.Items[i].Path == path && j.Items[i].State != "completed" && j.Items[i].State != "failed" {
			j.Items[i].State = state
			j.Items[i].Progress = progress
			j.Items[i].Output = output
			j.Items[i].Error = errText
			j.Current = path
			item := j.Items[i]
			snapshot := cloneJobStatus(j.JobStatus)
			a.mu.Unlock()
			a.emit("job:item", item)
			a.emit("job:progress", snapshot)
			return
		}
	}
	a.mu.Unlock()
}

func (a *App) update(j *job, fn func(*JobStatus)) {
	a.mu.Lock()
	fn(&j.JobStatus)
	snapshot := cloneJobStatus(j.JobStatus)
	a.mu.Unlock()
	a.emit("job:progress", snapshot)
}

func (a *App) finish(j *job) {
	now := time.Now()
	a.update(j, func(s *JobStatus) { s.FinishedAt = &now; s.Current = "" })
}

func cloneJobItems(items []JobItem) []JobItem {
	if items == nil {
		return nil
	}
	out := make([]JobItem, len(items))
	for i, item := range items {
		out[i] = item
		out[i].Outputs = append([]string(nil), item.Outputs...)
	}
	return out
}

func cloneJobStatus(status JobStatus) JobStatus {
	status.Items = cloneJobItems(status.Items)
	status.Outputs = append([]string(nil), status.Outputs...)
	return status
}

func (a *App) snapshot(j *job) JobStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return cloneJobStatus(j.JobStatus)
}

func (a *App) GetJob(id string) (*JobStatus, error) {
	a.mu.RLock()
	j, ok := a.jobs[id]
	a.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("job %q not found", id)
	}
	snapshot := a.snapshot(j)
	return &snapshot, nil
}

func (a *App) CancelJob(id string) error {
	a.mu.RLock()
	j, ok := a.jobs[id]
	a.mu.RUnlock()
	if !ok {
		return fmt.Errorf("job %q not found", id)
	}
	j.cancel()
	return nil
}

func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	info, err := a.updater.Check(a.version)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	if info == nil {
		a.lastUpdate = nil
	} else {
		copyInfo := *info
		copyInfo.Assets = append([]platform.UpdateAsset(nil), info.Assets...)
		a.lastUpdate = &copyInfo
	}
	a.mu.Unlock()
	return info, nil
}

func (a *App) DownloadAndInstallUpdate(assetID string) error {
	a.emit("update:progress", map[string]any{"assetId": assetID, "state": "started", "progress": 0})
	a.mu.RLock()
	if a.lastUpdate == nil {
		a.mu.RUnlock()
		a.emit("update:progress", map[string]any{"assetId": assetID, "state": "failed", "progress": 0})
		return errors.New("no update check has completed")
	}
	info := *a.lastUpdate
	info.Assets = append([]platform.UpdateAsset(nil), a.lastUpdate.Assets...)
	a.mu.RUnlock()
	if strings.TrimSpace(assetID) != "" && info.AssetID != assetID {
		for _, asset := range info.Assets {
			if asset.ID == assetID {
				info.AssetID = asset.ID
				info.URL = asset.URL
				info.SHA256 = asset.SHA256
				info.Signature = asset.Signature
				break
			}
		}
	}
	if strings.TrimSpace(assetID) != "" && info.AssetID != assetID {
		a.emit("update:progress", map[string]any{"assetId": assetID, "state": "failed", "progress": 0})
		return fmt.Errorf("update asset %q is not available", assetID)
	}
	err := a.updater.DownloadAndInstall(&info)
	state := "completed"
	if err != nil {
		state = "failed"
	}
	a.emit("update:progress", map[string]any{"assetId": assetID, "state": state, "progress": 1})
	return err
}
