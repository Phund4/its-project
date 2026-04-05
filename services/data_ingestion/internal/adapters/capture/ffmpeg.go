package capture

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// FFmpegPipe запускает ffmpeg: декодирует inputURL (RTSP, файл и т.д.) и пишет MJPEG в stdout.
// Вызывающий код обязан вызвать Close() у возвращённого ReadCloser, чтобы дождаться процесса.
func FFmpegPipe(ctx context.Context, ffmpegPath, inputURL string, fps float64) (io.ReadCloser, error) {
	fpsStr := strconv.FormatFloat(fps, 'f', -1, 64)
	args := []string{
		"-hide_banner",
		"-nostats",
		"-loglevel", "quiet",
	}
	u := strings.ToLower(strings.TrimSpace(inputURL))
	if strings.HasPrefix(u, "rtsp://") || strings.HasPrefix(u, "rtsps://") {
		args = append(args, "-rtsp_transport", "tcp")
	}
	args = append(args,
		"-i", inputURL,
		"-an",
		"-vf", "fps="+fpsStr,
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-q:v", "5",
		"-",
	)
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	cmd.Stderr = io.Discard
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdout.Close()
		return nil, fmt.Errorf("ffmpeg start: %w", err)
	}
	return &ffmpegReader{stdout: stdout, cmd: cmd}, nil
}

type ffmpegReader struct {
	stdout io.ReadCloser
	cmd    *exec.Cmd
}

// Read читает байты из stdout процесса ffmpeg.
func (f *ffmpegReader) Read(p []byte) (int, error) {
	return f.stdout.Read(p)
}

// Close закрывает stdout и ждёт завершения процесса.
func (f *ffmpegReader) Close() error {
	_ = f.stdout.Close()
	return f.cmd.Wait()
}
