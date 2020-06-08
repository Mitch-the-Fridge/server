package ffmpeg

import (
	"os/exec"
	"strconv"
)

const (
	video_codec = "libx264"
)

// MakeVideo creates a mp4 video from the given frames in the given dir and
// returns the path to the video. Removing the frames in the process.
func MakeVideo(inputDir, ext, output string, fps int) error {
	cmd := exec.Command(
		"nice",
		"-n5",

		"ffmpeg",
		"-y",
		"-r",
		strconv.Itoa(fps),
		"-i",
		"./%10d"+ext,
		"-threads",
		"0",
		"-c:v",
		video_codec,
		output,
	)
	cmd.Dir = inputDir
	return cmd.Run()
}
