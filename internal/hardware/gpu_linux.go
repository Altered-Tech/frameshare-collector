package hardware

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
)

var gpuClassPrefixes = []string{"VGA compatible controller", "3D controller", "Display controller"}

func collectGPUs(ctx context.Context) ([]GPUInfo, error) {
	cmd := exec.CommandContext(ctx, "lspci")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var gpus []GPUInfo
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		colon := strings.Index(line, ": ")
		if colon == -1 {
			continue
		}
		class := line[:colon]
		desc := line[colon+2:]

		isGPU := false
		for _, prefix := range gpuClassPrefixes {
			if strings.Contains(class, prefix) {
				isGPU = true
				break
			}
		}
		if !isGPU {
			continue
		}

		vendor := ""
		for _, v := range []string{"NVIDIA", "AMD", "Intel", "ATI"} {
			if strings.Contains(desc, v) {
				vendor = v
				break
			}
		}
		gpus = append(gpus, GPUInfo{
			Name:   strings.TrimSpace(desc),
			Vendor: vendor,
		})
	}
	return gpus, scanner.Err()
}
