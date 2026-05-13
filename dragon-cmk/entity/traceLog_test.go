package entity

import (
	"context"
	"testing"

	"github.com/PointerByte/GoForge/logger/builder"
	viperdata "github.com/PointerByte/GoForge/logger/viperData"
	"github.com/spf13/viper"
)

func TestTraceClient(t *testing.T) {
	viper.Reset()
	viperdata.ResetViperDataSingleton()
	t.Cleanup(func() {
		viper.Reset()
		viperdata.ResetViperDataSingleton()
	})

	viper.Set(string(viperdata.AppAtribute), "dragon-cmk")
	builder.EnableModeTest()
	ctxLogger := builder.New(context.Background())

	process, traceEnd := TraceClient(ctxLogger, "test process")
	if process == nil {
		t.Fatal("expected process to be created")
	}
	if traceEnd == nil {
		t.Fatal("expected trace end handler")
	}
	if process.System != "dragon-cmk" {
		t.Fatalf("unexpected system: %q", process.System)
	}
	if process.Process != "test process" {
		t.Fatalf("unexpected process name: %q", process.Process)
	}

	traceEnd(process)
}
