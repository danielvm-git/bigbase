package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ManifestProfile defines the standard build and run configuration for a runtime.
type ManifestProfile struct {
	AppType  AppType
	BuildCmd string
	RunCmd   string
	Port     int
}

// DefaultProfile returns the standard bigbase.toml profile defaults for a given runtime.
func DefaultProfile(appType AppType) ManifestProfile {
	switch appType {
	case AppRust:
		return ManifestProfile{
			AppType:  AppRust,
			BuildCmd: "cargo build --release",
			RunCmd:   "./target/release/app",
			Port:     8080,
		}
	case AppPHP:
		return ManifestProfile{
			AppType:  AppPHP,
			BuildCmd: "composer install --no-dev",
			RunCmd:   "php -S 0.0.0.0:$PORT",
			Port:     8080,
		}
	case AppJava:
		return ManifestProfile{
			AppType:  AppJava,
			BuildCmd: "mvn package -DskipTests",
			RunCmd:   "java -jar target/app.jar",
			Port:     8080,
		}
	case AppRuby:
		return ManifestProfile{
			AppType:  AppRuby,
			BuildCmd: "bundle install --deployment",
			RunCmd:   "bundle exec ruby app.rb",
			Port:     8080,
		}
	case AppGo:
		return ManifestProfile{
			AppType:  AppGo,
			BuildCmd: "go build -o app .",
			RunCmd:   "./app",
			Port:     8080,
		}
	case AppNode:
		return ManifestProfile{
			AppType:  AppNode,
			BuildCmd: "npm install && npm run build",
			RunCmd:   "npm start",
			Port:     3000,
		}
	case AppPython:
		return ManifestProfile{
			AppType:  AppPython,
			BuildCmd: "pip install -r requirements.txt",
			RunCmd:   "python main.py",
			Port:     8000,
		}
	default:
		return ManifestProfile{
			AppType:  AppStatic,
			BuildCmd: "",
			RunCmd:   "",
			Port:     80,
		}
	}
}

// RustBuildCommand returns the command and arguments for building a Rust application.
func RustBuildCommand(dir string) (string, []string) {
	return "cargo", []string{"build", "--release"}
}

// RustStartCommand returns the exec.Cmd for starting a compiled Rust binary.
func RustStartCommand(ctx context.Context, dir string) *exec.Cmd {
	binPath := rustBinaryPath(dir)
	cmd := exec.CommandContext(ctx, binPath)
	cmd.Dir = dir
	return cmd
}

func rustBinaryPath(dir string) string {
	releaseDir := filepath.Join(dir, "target", "release")
	if entries, err := os.ReadDir(releaseDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(name, ".d") || strings.HasSuffix(name, ".rlib") || strings.HasPrefix(name, ".") {
				continue
			}
			info, err := e.Info()
			if err == nil && (info.Mode()&0111 != 0 || !strings.Contains(name, ".")) {
				return filepath.Join(releaseDir, name)
			}
		}
	}
	return filepath.Join(releaseDir, "app")
}

// PHPBuildCommand returns the command and arguments for installing PHP dependencies.
func PHPBuildCommand(dir string) (string, []string) {
	return "composer", []string{"install", "--no-dev", "--optimize-autoloader", "--no-interaction"}
}

// PHPStartCommand returns the exec.Cmd for running PHP with the built-in web server.
func PHPStartCommand(ctx context.Context, dir string, port int) *exec.Cmd {
	docRoot := dir
	if st, err := os.Stat(filepath.Join(dir, "public")); err == nil && st.IsDir() {
		docRoot = filepath.Join(dir, "public")
	}
	cmd := exec.CommandContext(ctx, "php", "-S", fmt.Sprintf("0.0.0.0:%d", port), "-t", docRoot)
	cmd.Dir = dir
	return cmd
}

// JavaBuildCommand returns the command and arguments for building a Java application.
func JavaBuildCommand(dir string) (string, []string) {
	if fileExists(filepath.Join(dir, "pom.xml")) {
		return "mvn", []string{"package", "-DskipTests"}
	}
	if fileExists(filepath.Join(dir, "gradlew")) {
		return "./gradlew", []string{"build", "-x", "test"}
	}
	return "gradle", []string{"build", "-x", "test"}
}

// JavaStartCommand returns the exec.Cmd for running a compiled Java JAR.
func JavaStartCommand(ctx context.Context, dir string) *exec.Cmd {
	jarPath := javaJarPath(dir)
	cmd := exec.CommandContext(ctx, "java", "-jar", jarPath)
	cmd.Dir = dir
	return cmd
}

func javaJarPath(dir string) string {
	if fileExists(filepath.Join(dir, "app.jar")) {
		return filepath.Join(dir, "app.jar")
	}
	targetDir := filepath.Join(dir, "target")
	if entries, err := os.ReadDir(targetDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".jar") &&
				!strings.HasSuffix(e.Name(), "-sources.jar") &&
				!strings.HasSuffix(e.Name(), "-javadoc.jar") &&
				!strings.HasSuffix(e.Name(), "-plain.jar") {
				return filepath.Join(targetDir, e.Name())
			}
		}
	}
	buildLibsDir := filepath.Join(dir, "build", "libs")
	if entries, err := os.ReadDir(buildLibsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".jar") &&
				!strings.HasSuffix(e.Name(), "-sources.jar") &&
				!strings.HasSuffix(e.Name(), "-javadoc.jar") &&
				!strings.HasSuffix(e.Name(), "-plain.jar") {
				return filepath.Join(buildLibsDir, e.Name())
			}
		}
	}
	return filepath.Join(targetDir, "app.jar")
}

// RubyBuildCommand returns the command and arguments for installing Ruby dependencies.
func RubyBuildCommand(dir string) (string, []string) {
	return "bundle", []string{"install", "--deployment"}
}

// RubyStartCommand returns the exec.Cmd for running a Ruby application.
func RubyStartCommand(ctx context.Context, dir string, port int) *exec.Cmd {
	var cmd *exec.Cmd
	if fileExists(filepath.Join(dir, "config.ru")) {
		cmd = exec.CommandContext(ctx, "bundle", "exec", "rackup", "-p", fmt.Sprintf("%d", port), "-o", "0.0.0.0")
	} else {
		cmd = exec.CommandContext(ctx, "bundle", "exec", "ruby", "app.rb")
	}
	cmd.Dir = dir
	return cmd
}
