package mediainfo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	VideoPresetH2641080p = "h264-1080p"
	VideoPresetH264720p  = "h264-720p"
	VideoPresetHEVC1080p = "hevc-1080p"
	VideoPresetHEVC720p  = "hevc-720p"
)

type VideoInfo struct {
	Width     int
	Height    int
	Duration  float64
	Codec     string
	SizeBytes int64
}

// ProbeVideo inspects video dimensions, codec, duration, and file size.
// If ffprobe is not installed, it still returns SizeBytes cleanly.
func ProbeVideo(path string) (VideoInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return VideoInfo{}, err
	}
	info := VideoInfo{SizeBytes: stat.Size()}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return info, nil
	}
	out, err := exec.Command(ffprobe,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height,codec_name:format=duration",
		"-of", "json",
		path,
	).Output()
	if err != nil {
		return info, nil
	}
	var parsed struct {
		Streams []struct {
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			CodecName string `json:"codec_name"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if json.Unmarshal(out, &parsed) == nil {
		if len(parsed.Streams) > 0 {
			info.Width = parsed.Streams[0].Width
			info.Height = parsed.Streams[0].Height
			info.Codec = strings.ToUpper(parsed.Streams[0].CodecName)
		}
		if d, err := strconv.ParseFloat(strings.TrimSpace(parsed.Format.Duration), 64); err == nil {
			info.Duration = d
		}
	}
	return info, nil
}

// CompressVideo transcodes srcPath to dstPath using an H.264/HEVC preset via
// ffmpeg, preferring Windows Media Foundation hardware encoders (h264_mf /
// hevc_mf) with automatic software fallback (libx264 / libx265).
func CompressVideo(ctx context.Context, srcPath, dstPath, preset string) error {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return errors.New("未检测到 ffmpeg，请安装 ffmpeg 并加入 PATH 后再压缩视频")
	}
	var hwCodec, swCodec, scaleFilter, bitrate string
	switch preset {
	case VideoPresetH2641080p:
		hwCodec, swCodec, scaleFilter, bitrate = "h264_mf", "libx264", "scale='min(1920,iw)':-2", "4500k"
	case VideoPresetH264720p:
		hwCodec, swCodec, scaleFilter, bitrate = "h264_mf", "libx264", "scale='min(1280,iw)':-2", "2200k"
	case VideoPresetHEVC1080p:
		hwCodec, swCodec, scaleFilter, bitrate = "hevc_mf", "libx265", "scale='min(1920,iw)':-2", "3000k"
	case VideoPresetHEVC720p:
		hwCodec, swCodec, scaleFilter, bitrate = "hevc_mf", "libx265", "scale='min(1280,iw)':-2", "1500k"
	default:
		return fmt.Errorf("不支持的视频压缩预设：%s", preset)
	}

	run := func(codec string, extra []string) error {
		args := []string{
			"-y", "-v", "error",
			"-i", srcPath,
			"-vf", scaleFilter,
			"-c:v", codec,
		}
		args = append(args, extra...)
		args = append(args, "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", dstPath)
		return exec.CommandContext(ctx, ffmpeg, args...).Run()
	}

	if err := run(hwCodec, []string{"-b:v", bitrate}); err == nil {
		return nil
	}
	if err := run(swCodec, []string{"-crf", "24", "-preset", "medium"}); err != nil {
		return fmt.Errorf("视频压缩失败：%w", err)
	}
	return nil
}
