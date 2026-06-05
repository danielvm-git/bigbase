package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielvm/bigbase/components/deploy"
)

func BenchmarkDeployDetectAppType(b *testing.B) {
	dir := b.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{}`), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deploy.DetectAppType(dir)
	}
}

func BenchmarkDeployGetStartCommand(b *testing.B) {
	dir := b.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"start":"node app.js"}}`), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = deploy.GetStartCommand(dir)
	}
}
