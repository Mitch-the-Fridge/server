package main

import (
	"os/exec"
	"strconv"
)

const (
	codec = "libx264"
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
		codec,
		output,
	)
	cmd.Dir = inputDir
	return cmd.Run()
}
