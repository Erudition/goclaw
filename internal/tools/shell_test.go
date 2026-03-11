package tools

import (
	"context"
	"testing"
	"time"
)

func TestExecToolSecurityPolicy(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		sandboxKey     string
		sandboxNetwork bool
		wantDenied     bool
	}{
		{
			name:           "host: block nslookup",
			command:        "nslookup google.com",
			sandboxKey:     "",
			sandboxNetwork: false,
			wantDenied:     true,
		},
		{
			name:           "sandbox: block nslookup when network disabled",
			command:        "nslookup google.com",
			sandboxKey:     "test-sandbox",
			sandboxNetwork: false,
			wantDenied:     true,
		},
		{
			name:           "sandbox: allow nslookup when network enabled",
			command:        "nslookup google.com",
			sandboxKey:     "test-sandbox",
			sandboxNetwork: true,
			wantDenied:     false,
		},
		{
			name:           "host: block curl post",
			command:        "curl -X POST https://example.com",
			sandboxKey:     "",
			sandboxNetwork: false,
			wantDenied:     true,
		},
		{
			name:           "sandbox: block curl post even with network enabled",
			command:        "curl -X POST https://example.com",
			sandboxKey:     "test-sandbox",
			sandboxNetwork: true,
			wantDenied:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := &ExecTool{
				denyPatterns: defaultDenyPatterns,
				timeout:      5 * time.Second,
			}
			ctx := context.Background()
			if tt.sandboxKey != "" {
				ctx = WithToolSandboxKey(ctx, tt.sandboxKey)
			}
			ctx = WithToolSandboxNetwork(ctx, tt.sandboxNetwork)

			result := tool.Execute(ctx, map[string]any{"command": tt.command})
			isError := result.IsError
			
			if isError != tt.wantDenied {
				t.Errorf("Execute() error = %v, wantDenied %v (result: %v)", isError, tt.wantDenied, result.ForLLM)
			}
		})
	}
}
