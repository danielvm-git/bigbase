package deploy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectRust(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"foo\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := DetectAppType(dir); got != AppRust {
		t.Fatalf("expected AppRust, got %s", got)
	}
}

func TestBuildRust(t *testing.T) {
	dir := t.TempDir()
	cmd, args := RustBuildCommand(dir)
	if cmd != "cargo" {
		t.Errorf("RustBuildCommand cmd = %q, want cargo", cmd)
	}
	wantArgs := []string{"build", "--release"}
	if strings.Join(args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("RustBuildCommand args = %v, want %v", args, wantArgs)
	}
}

func TestRunRust(t *testing.T) {
	dir := t.TempDir()
	releaseDir := filepath.Join(dir, "target", "release")
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		t.Fatal(err)
	}
	binFile := filepath.Join(releaseDir, "myservice")
	if err := os.WriteFile(binFile, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	cmd := RustStartCommand(context.Background(), dir)
	if cmd == nil {
		t.Fatal("expected non-nil cmd from RustStartCommand")
	}
	if cmd.Path != binFile && !strings.HasSuffix(cmd.Path, "myservice") {
		t.Errorf("RustStartCommand path = %q, want %q", cmd.Path, binFile)
	}
}

func TestDetectPHP(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"name":"app/php"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := DetectAppType(dir); got != AppPHP {
		t.Fatalf("expected AppPHP, got %s", got)
	}
}

func TestBuildPHP(t *testing.T) {
	dir := t.TempDir()
	cmd, args := PHPBuildCommand(dir)
	if cmd != "composer" {
		t.Errorf("PHPBuildCommand cmd = %q, want composer", cmd)
	}
	wantArgs := []string{"install", "--no-dev", "--optimize-autoloader", "--no-interaction"}
	if strings.Join(args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("PHPBuildCommand args = %v, want %v", args, wantArgs)
	}
}

func TestRunPHP(t *testing.T) {
	t.Run("with public docroot", func(t *testing.T) {
		dir := t.TempDir()
		publicDir := filepath.Join(dir, "public")
		if err := os.MkdirAll(publicDir, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := PHPStartCommand(context.Background(), dir, 8080)
		if cmd == nil {
			t.Fatal("expected non-nil cmd from PHPStartCommand")
		}
		foundPort := false
		foundDocRoot := false
		for _, arg := range cmd.Args {
			if arg == "0.0.0.0:8080" {
				foundPort = true
			}
			if arg == publicDir {
				foundDocRoot = true
			}
		}
		if !foundPort {
			t.Errorf("expected 0.0.0.0:8080 in PHP args: %v", cmd.Args)
		}
		if !foundDocRoot {
			t.Errorf("expected public dir in PHP args: %v", cmd.Args)
		}
	})

	t.Run("without public docroot", func(t *testing.T) {
		dir := t.TempDir()
		cmd := PHPStartCommand(context.Background(), dir, 8080)
		if cmd == nil {
			t.Fatal("expected non-nil cmd from PHPStartCommand")
		}
		foundDocRoot := false
		for _, arg := range cmd.Args {
			if arg == dir {
				foundDocRoot = true
			}
		}
		if !foundDocRoot {
			t.Errorf("expected base dir in PHP args: %v", cmd.Args)
		}
	})
}

func TestDetectJava(t *testing.T) {
	t.Run("pom.xml", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project></project>"), 0644); err != nil {
			t.Fatal(err)
		}
		if got := DetectAppType(dir); got != AppJava {
			t.Fatalf("expected AppJava for pom.xml, got %s", got)
		}
	})

	t.Run("build.gradle", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("plugins {}"), 0644); err != nil {
			t.Fatal(err)
		}
		if got := DetectAppType(dir); got != AppJava {
			t.Fatalf("expected AppJava for build.gradle, got %s", got)
		}
	})

	t.Run("build.gradle.kts", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte("plugins {}"), 0644); err != nil {
			t.Fatal(err)
		}
		if got := DetectAppType(dir); got != AppJava {
			t.Fatalf("expected AppJava for build.gradle.kts, got %s", got)
		}
	})
}

func TestBuildJava(t *testing.T) {
	t.Run("maven", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0644)
		cmd, args := JavaBuildCommand(dir)
		if cmd != "mvn" {
			t.Errorf("JavaBuildCommand maven cmd = %q, want mvn", cmd)
		}
		want := "package -DskipTests"
		if strings.Join(args, " ") != want {
			t.Errorf("JavaBuildCommand maven args = %v, want %v", args, want)
		}
	})

	t.Run("gradlew", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(""), 0644)
		_ = os.WriteFile(filepath.Join(dir, "gradlew"), []byte("#!/bin/sh\n"), 0755)
		cmd, args := JavaBuildCommand(dir)
		if cmd != "./gradlew" {
			t.Errorf("JavaBuildCommand gradlew cmd = %q, want ./gradlew", cmd)
		}
		want := "build -x test"
		if strings.Join(args, " ") != want {
			t.Errorf("JavaBuildCommand gradlew args = %v, want %v", args, want)
		}
	})

	t.Run("gradle", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(""), 0644)
		cmd, args := JavaBuildCommand(dir)
		if cmd != "gradle" {
			t.Errorf("JavaBuildCommand gradle cmd = %q, want gradle", cmd)
		}
		want := "build -x test"
		if strings.Join(args, " ") != want {
			t.Errorf("JavaBuildCommand gradle args = %v, want %v", args, want)
		}
	})
}

func TestRunJava(t *testing.T) {
	t.Run("target jar", func(t *testing.T) {
		dir := t.TempDir()
		targetDir := filepath.Join(dir, "target")
		_ = os.MkdirAll(targetDir, 0755)
		jarFile := filepath.Join(targetDir, "myservice.jar")
		_ = os.WriteFile(jarFile, []byte("PK"), 0644)

		cmd := JavaStartCommand(context.Background(), dir)
		if cmd == nil {
			t.Fatal("expected non-nil cmd from JavaStartCommand")
		}
		foundJar := false
		for _, arg := range cmd.Args {
			if arg == jarFile {
				foundJar = true
			}
		}
		if !foundJar {
			t.Errorf("expected jar path in java args: %v", cmd.Args)
		}
	})

	t.Run("build/libs jar", func(t *testing.T) {
		dir := t.TempDir()
		libsDir := filepath.Join(dir, "build", "libs")
		_ = os.MkdirAll(libsDir, 0755)
		jarFile := filepath.Join(libsDir, "myservice.jar")
		_ = os.WriteFile(jarFile, []byte("PK"), 0644)

		cmd := JavaStartCommand(context.Background(), dir)
		if cmd == nil {
			t.Fatal("expected non-nil cmd from JavaStartCommand")
		}
		foundJar := false
		for _, arg := range cmd.Args {
			if arg == jarFile {
				foundJar = true
			}
		}
		if !foundJar {
			t.Errorf("expected jar path in java args: %v", cmd.Args)
		}
	})

	t.Run("skips plain jar", func(t *testing.T) {
		dir := t.TempDir()
		libsDir := filepath.Join(dir, "build", "libs")
		_ = os.MkdirAll(libsDir, 0755)
		_ = os.WriteFile(filepath.Join(libsDir, "app-0.0.1-plain.jar"), []byte("PK"), 0644)
		_ = os.WriteFile(filepath.Join(libsDir, "app-0.0.1.jar"), []byte("PK"), 0644)

		cmd := JavaStartCommand(context.Background(), dir)
		if cmd == nil {
			t.Fatal("expected non-nil cmd from JavaStartCommand")
		}
		wantJar := filepath.Join(libsDir, "app-0.0.1.jar")
		if cmd.Args[2] != wantJar {
			t.Errorf("got %s, want %s (must skip -plain.jar)", cmd.Args[2], wantJar)
		}
	})
}

func TestDetectRuby(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := DetectAppType(dir); got != AppRuby {
		t.Fatalf("expected AppRuby, got %s", got)
	}
}

func TestBuildRuby(t *testing.T) {
	dir := t.TempDir()
	cmd, args := RubyBuildCommand(dir)
	if cmd != "bundle" {
		t.Errorf("RubyBuildCommand cmd = %q, want bundle", cmd)
	}
	want := "install --deployment"
	if strings.Join(args, " ") != want {
		t.Errorf("RubyBuildCommand args = %v, want %v", args, want)
	}
}

func TestRunRuby(t *testing.T) {
	t.Run("with config.ru", func(t *testing.T) {
		dir := t.TempDir()
		_ = os.WriteFile(filepath.Join(dir, "config.ru"), []byte("run App\n"), 0644)
		cmd := RubyStartCommand(context.Background(), dir, 8080)
		if cmd == nil {
			t.Fatal("expected non-nil cmd from RubyStartCommand")
		}
		foundRackup := false
		for _, arg := range cmd.Args {
			if arg == "rackup" {
				foundRackup = true
			}
		}
		if !foundRackup {
			t.Errorf("expected rackup in ruby args: %v", cmd.Args)
		}
	})

	t.Run("with app.rb default", func(t *testing.T) {
		dir := t.TempDir()
		cmd := RubyStartCommand(context.Background(), dir, 8080)
		if cmd == nil {
			t.Fatal("expected non-nil cmd from RubyStartCommand")
		}
		foundAppRb := false
		for _, arg := range cmd.Args {
			if arg == "app.rb" {
				foundAppRb = true
			}
		}
		if !foundAppRb {
			t.Errorf("expected app.rb in ruby args: %v", cmd.Args)
		}
	})
}

func TestAllAppTypes(t *testing.T) {
	expected := []AppType{
		AppNode,
		AppGo,
		AppPython,
		AppPHP,
		AppRust,
		AppJava,
		AppRuby,
		AppStatic,
		AppStaticSidecar,
	}

	all := AllAppTypes()
	if len(all) != len(expected) {
		t.Fatalf("AllAppTypes() len = %d, want %d", len(all), len(expected))
	}

	for _, at := range expected {
		found := false
		for _, a := range all {
			if a == at {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected AppType %q in AllAppTypes()", at)
		}
		if !at.IsValid() {
			t.Errorf("expected AppType %q to be valid", at)
		}
	}
}

func TestManifestProfiles(t *testing.T) {
	runtimes := []AppType{
		AppRust,
		AppPHP,
		AppJava,
		AppRuby,
		AppGo,
		AppNode,
		AppPython,
	}

	for _, rt := range runtimes {
		profile := DefaultProfile(rt)
		if profile.AppType != rt {
			t.Errorf("profile.AppType = %q, want %q", profile.AppType, rt)
		}
		if profile.BuildCmd == "" {
			t.Errorf("profile.BuildCmd is empty for %q", rt)
		}
		if profile.RunCmd == "" {
			t.Errorf("profile.RunCmd is empty for %q", rt)
		}
		if profile.Port <= 0 {
			t.Errorf("profile.Port = %d for %q, want > 0", profile.Port, rt)
		}
	}

	// Test manifest framework parsing for each new runtime
	manifestTests := []struct {
		toml string
		want AppType
	}{
		{`version = 1
framework = "rust"
[build]
command = "cargo build --release"
[start]
command = "./target/release/app"
port = 8080
`, AppRust},
		{`version = 1
framework = "php"
[build]
command = "composer install --no-dev"
[start]
command = "php -S 0.0.0.0:8080"
port = 8080
`, AppPHP},
		{`version = 1
framework = "java"
[build]
command = "mvn package -DskipTests"
[start]
command = "java -jar target/app.jar"
port = 8080
`, AppJava},
		{`version = 1
framework = "ruby"
[build]
command = "bundle install --deployment"
[start]
command = "bundle exec ruby app.rb"
port = 8080
`, AppRuby},
	}

	for _, mt := range manifestTests {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "bigbase.toml"), []byte(mt.toml), 0644); err != nil {
			t.Fatal(err)
		}
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatalf("LoadManifest(%q): %v", mt.toml, err)
		}
		if got := manifestToAppType(m); got != mt.want {
			t.Errorf("manifestToAppType = %q, want %q for %s", got, mt.want, mt.toml)
		}
	}
}
