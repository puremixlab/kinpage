package nativeui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestNativeUIPhase6TipCreatorSheet(t *testing.T) {
	fb := &fakeBackend{}
	app := New(fb)
	var openedURI string
	app.openExternal = func(u string) error {
		openedURI = u
		return nil
	}

	tt := ui.NewTester(app.View, 1280, 820)
	tt.Frame()

	app.openTipModal("planetable.eth")
	tt.Frame()
	if err := tt.Click("0.01 ETH"); err != nil {
		t.Fatalf("expected 0.01 ETH preset button: %v", err)
	}
	tt.Frame()
	if err := tt.Click("在钱包中打开支付链接"); err != nil {
		t.Fatalf("expected primary tip button: %v", err)
	}
	tt.Frame()
	if openedURI != "ethereum:planetable.eth@1?value=10000000000000000" {
		t.Fatalf("unexpected tip URI: %q", openedURI)
	}
	if app.tipModalOpen {
		t.Fatal("expected tip sheet to close after opening wallet link")
	}
}
