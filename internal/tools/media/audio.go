package media

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func GetAudioDuration(audio []byte) (uint32, error) {
	filename := fmt.Sprintf("%s/%d", tempDir, time.Now().UnixNano())
	err := os.WriteFile(filename, audio, 0644)
	if err != nil {
		return 0, err
	}
	defer os.Remove(filename)
	cmd := exec.Command("ffprobe", "-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		filename)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseUint(durationStr, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(duration), nil
}
