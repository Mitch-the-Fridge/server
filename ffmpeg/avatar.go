package ffmpeg

import (
	"os/exec"
)

const (
	avatar_codec     = "libx264"
	avatar_max_width = "200" // in pixels
)

func MakeAvatar(inputVideo, output string) error {
	cmd := exec.Command(
		"nice",
		"-n5",

		"ffmpeg",
		"-y",
		"-i",
		inputVideo,
		"-vf",
		"crop=min(in_w\\,in_h):min(in_w\\,in_h),scale="+avatar_max_width+":-1",
		"-threads",
		"0",
		"-c:v",
		avatar_codec,
		output,
	)
	return cmd.Run()
}
