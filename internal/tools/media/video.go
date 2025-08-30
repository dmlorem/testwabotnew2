package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func GetVideoThumbnail(video []byte) (videoThumbnail []byte, err error) {
	timestamp := time.Now().UnixNano()
	tempVideoPath := filepath.Join(tempDir, fmt.Sprintf("video_%d.mp4", timestamp))
	outputFilePath := filepath.Join(tempDir, fmt.Sprintf("image_%d.jpg", timestamp))

	if err = os.WriteFile(tempVideoPath, video, 0644); err != nil {
		return videoThumbnail, err
	}
	defer os.Remove(tempVideoPath)

	cmd := exec.Command("ffmpeg", "-i", tempVideoPath, "-ss", "00:00:00", "-vf", "scale=32:-1", "-vframes", "1", "-f", "image2", outputFilePath)
	if err = cmd.Run(); err != nil {
		return videoThumbnail, err
	}
	defer os.Remove(outputFilePath)

	return os.ReadFile(outputFilePath)
}
