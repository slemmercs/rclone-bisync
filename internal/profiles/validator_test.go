package profiles

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  []string
		expectErr bool
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "simple flags",
			input:    "--bwlimit 10M --transfers 4",
			expected: []string{"--bwlimit", "10M", "--transfers", "4"},
		},
		{
			name:     "double quotes with space",
			input:    `--filter "+ /Documents/**" --checkers 8`,
			expected: []string{"--filter", "+ /Documents/**", "--checkers", "8"},
		},
		{
			name:     "single quotes with special chars",
			input:    `--exclude '*.tmp' --fast-list`,
			expected: []string{"--exclude", "*.tmp", "--fast-list"},
		},
		{
			name:     "escaped space",
			input:    `--tag my\ tag --verbose`,
			expected: []string{"--tag", "my tag", "--verbose"},
		},
		{
			name:     "flag with equals",
			input:    `--bwlimit=15M --tpslimit=10`,
			expected: []string{"--bwlimit=15M", "--tpslimit=10"},
		},
		{
			name:      "unclosed double quote",
			input:     `--filter "unclosed`,
			expectErr: true,
		},
		{
			name:      "unclosed single quote",
			input:     `--filter 'unclosed`,
			expectErr: true,
		},
		{
			name:      "trailing escape",
			input:     `--flag test\`,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitCommandLine(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("got %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestValidateCustomFlags(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid custom flags",
			input:     "--bwlimit 10M --transfers 4 --fast-list",
			expectErr: false,
		},
		{
			name:      "empty custom flags",
			input:     "   ",
			expectErr: false,
		},
		{
			name:      "prohibited --config",
			input:     "--config /etc/rclone.conf",
			expectErr: true,
			errMsg:    "prohibited custom flag '--config'",
		},
		{
			name:      "prohibited -c",
			input:     "-c /custom/path",
			expectErr: true,
			errMsg:    "prohibited custom flag '-c'",
		},
		{
			name:      "prohibited --rc with equals",
			input:     "--rc=true",
			expectErr: true,
			errMsg:    "prohibited custom flag '--rc'",
		},
		{
			name:      "prohibited --log-file",
			input:     "--transfers 2 --log-file /tmp/rclone.log",
			expectErr: true,
			errMsg:    "prohibited custom flag '--log-file'",
		},
		{
			name:      "syntax error unclosed quote",
			input:     `--bwlimit "10M`,
			expectErr: true,
			errMsg:    "unclosed quote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := ValidateCustomFlags(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.TrimSpace(tt.input) != "" && len(tokens) == 0 {
				t.Errorf("expected non-empty tokens for input: %q", tt.input)
			}
		})
	}
}
