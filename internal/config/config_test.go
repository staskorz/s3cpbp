package config

import (
	"flag"
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	// Save original flag values and restore them after test
	oldFlagCommandLine := flag.CommandLine
	defer func() { flag.CommandLine = oldFlagCommandLine }()

	// Reset flags for each test case
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	tests := []struct {
		name          string
		args          []string
		version       string
		expectedCfg   *Config
		expectVersion bool
		wantErr       bool
		tempDir       bool
	}{
		{
			name:    "full command with long flags",
			args:    []string{"-bucket", "test-bucket", "-prefix", "test-prefix", "-destination", "test-dest", "-concurrency", "5"},
			version: "1.0.0",
			expectedCfg: &Config{
				Bucket:      "test-bucket",
				Prefix:      "test-prefix",
				Destination: "test-dest",
				Concurrency: 5,
				Version:     "1.0.0",
			},
			expectVersion: false,
			wantErr:       false,
		},
		{
			name:    "full command with short flags",
			args:    []string{"-b", "test-bucket", "-p", "test-prefix", "-d", "test-dest", "-c", "5"},
			version: "1.0.0",
			expectedCfg: &Config{
				Bucket:      "test-bucket",
				Prefix:      "test-prefix",
				Destination: "test-dest",
				Concurrency: 5,
				Version:     "1.0.0",
			},
			expectVersion: false,
			wantErr:       false,
		},
		{
			name:          "version flag",
			args:          []string{"-version"},
			version:       "1.0.0",
			expectedCfg:   nil,
			expectVersion: true,
			wantErr:       false,
		},
		{
			name:          "version flag shorthand",
			args:          []string{"-v"},
			version:       "1.0.0",
			expectedCfg:   nil,
			expectVersion: true,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Create temp dir if needed
			var tempDir string
			if tt.tempDir {
				var err error
				tempDir, err = os.MkdirTemp("", "s3cpbp-test")
				if err != nil {
					t.Fatalf("failed to create temp dir: %v", err)
				}
				defer os.RemoveAll(tempDir)
				// Replace the destination path with the temp dir
				for i, arg := range tt.args {
					if arg == "-destination" || arg == "-d" {
						tt.args[i+1] = tempDir
						if tt.expectedCfg != nil {
							tt.expectedCfg.Destination = tempDir
						}
					}
				}
			}

			// Set up test
			os.Args = append([]string{"cmd"}, tt.args...)

			// Call function under test with custom exit function to prevent os.Exit
			var originalOsExit = osExit
			defer func() { osExit = originalOsExit }()

			// Use a panic to simulate os.Exit without actually exiting
			osExit = func(code int) {
				panic("os.Exit called")
			}

			// Setup recovery to catch the panic
			defer func() {
				if r := recover(); r != nil {
					if r != "os.Exit called" {
						t.Errorf("unexpected panic: %v", r)
					}
				}
			}()

			cfg, showVersion := Parse(tt.version)

			// Check results
			if tt.expectVersion != showVersion {
				t.Errorf("Parse() showVersion = %v, want %v", showVersion, tt.expectVersion)
			}

			if tt.expectedCfg != nil {
				if cfg == nil {
					t.Fatalf("Parse() returned nil Config, expected non-nil")
				}
				if cfg.Bucket != tt.expectedCfg.Bucket {
					t.Errorf("Parse() Bucket = %v, want %v", cfg.Bucket, tt.expectedCfg.Bucket)
				}
				if cfg.Prefix != tt.expectedCfg.Prefix {
					t.Errorf("Parse() Prefix = %v, want %v", cfg.Prefix, tt.expectedCfg.Prefix)
				}
				if cfg.Destination != tt.expectedCfg.Destination {
					t.Errorf("Parse() Destination = %v, want %v", cfg.Destination, tt.expectedCfg.Destination)
				}
				if cfg.Concurrency != tt.expectedCfg.Concurrency {
					t.Errorf("Parse() Concurrency = %v, want %v", cfg.Concurrency, tt.expectedCfg.Concurrency)
				}
				if cfg.Version != tt.expectedCfg.Version {
					t.Errorf("Parse() Version = %v, want %v", cfg.Version, tt.expectedCfg.Version)
				}
			}
		})
	}
}

// Redefine the osExit variable to allow testing fatal errors
var osExit = os.Exit
