package hardware

import (
	"bytes"
	"context"
	"os/exec"
)

// runPowerShellJSON runs a PowerShell command that pipes to ConvertTo-Json
// and returns the raw JSON bytes. ConvertTo-Json emits a bare object (not an
// array) when the pipeline has exactly one item, so callers must handle
// both shapes.
func runPowerShellJSON(ctx context.Context, script string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return bytes.TrimSpace(out.Bytes()), nil
}

// normalizeJSONArray wraps a bare JSON object in `[]` so single-result
// PowerShell ConvertTo-Json output can be unmarshaled the same way as
// multi-result output.
func normalizeJSONArray(data []byte) []byte {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return []byte("[]")
	}
	if trimmed[0] == '{' {
		return append(append([]byte("["), trimmed...), ']')
	}
	return trimmed
}
