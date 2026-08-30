package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/65658dsf/SimpleTools/internal/platform"
)

type fakeUpdateSource struct {
	info       *platform.UpdateInfo
	installErr error
	onInstall  func(*platform.UpdateInfo)
}

func (f *fakeUpdateSource) Check(string) (*platform.UpdateInfo, error) {
	return f.info, nil
}

func (f *fakeUpdateSource) DownloadAndInstall(info *platform.UpdateInfo) error {
	if f.onInstall != nil {
		f.onInstall(info)
	}
	return f.installErr
}

func TestDownloadAndInstallUpdateQuitsAfterCompletedEvent(t *testing.T) {
	const assetID = "windows-amd64-installer"
	steps := []string{}
	updater := &fakeUpdateSource{
		info: &platform.UpdateInfo{
			Available: true,
			AssetID:   assetID,
			URL:       "https://example.test/SimpleTools.exe",
		},
		onInstall: func(info *platform.UpdateInfo) {
			steps = append(steps, "install")
			if info.AssetID != assetID {
				t.Errorf("installed asset = %q, want %q", info.AssetID, assetID)
			}
		},
	}
	a := New()
	a.updater = updater
	if _, err := a.CheckForUpdate(); err != nil {
		t.Fatal(err)
	}

	ctx := context.WithValue(context.Background(), struct{}{}, "update-test")
	a.ctx = ctx
	a.setEventSink(func(name string, payload any) {
		if name != "update:progress" {
			return
		}
		steps = append(steps, "event:"+updateProgressState(t, payload))
	})
	a.quitAfterUpdate = func(got context.Context) {
		steps = append(steps, "quit")
		if got != ctx {
			t.Errorf("quit context does not match app context")
		}
	}

	if err := a.DownloadAndInstallUpdate(assetID); err != nil {
		t.Fatal(err)
	}
	want := []string{"event:started", "install", "event:completed", "quit"}
	if !reflect.DeepEqual(steps, want) {
		t.Fatalf("call order = %#v, want %#v", steps, want)
	}
}

func TestDownloadAndInstallUpdateDoesNotQuitWhenInstallerLaunchFails(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "launch failure", err: errors.New("start installer: executable file not found")},
		{name: "UAC cancelled", err: errors.New("start installer: the operation was canceled by the user")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const assetID = "windows-amd64-installer"
			steps := []string{}
			updater := &fakeUpdateSource{
				info: &platform.UpdateInfo{
					Available: true,
					AssetID:   assetID,
					URL:       "https://example.test/SimpleTools.exe",
				},
				installErr: tt.err,
				onInstall: func(*platform.UpdateInfo) {
					steps = append(steps, "install")
				},
			}
			a := New()
			a.updater = updater
			if _, err := a.CheckForUpdate(); err != nil {
				t.Fatal(err)
			}
			a.ctx = context.Background()
			a.setEventSink(func(name string, payload any) {
				if name == "update:progress" {
					steps = append(steps, "event:"+updateProgressState(t, payload))
				}
			})
			a.quitAfterUpdate = func(context.Context) {
				steps = append(steps, "quit")
			}

			err := a.DownloadAndInstallUpdate(assetID)
			if !errors.Is(err, tt.err) {
				t.Fatalf("error = %v, want %v", err, tt.err)
			}
			want := []string{"event:started", "install", "event:failed"}
			if !reflect.DeepEqual(steps, want) {
				t.Fatalf("call order = %#v, want %#v", steps, want)
			}
		})
	}
}

func updateProgressState(t *testing.T, payload any) string {
	t.Helper()
	progress, ok := payload.(map[string]any)
	if !ok {
		t.Fatalf("update progress payload type = %T", payload)
	}
	state, ok := progress["state"].(string)
	if !ok {
		t.Fatalf("update progress state = %#v", progress["state"])
	}
	return state
}
