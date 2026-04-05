package capture

import (
	"bytes"
	"context"
	"io"
	"sync"

	"data-ingestion/internal/constants"
)

// Scanner извлекает последовательные JPEG-кадры из MJPEG-потока (например, ffmpeg image2pipe).
type Scanner struct {
	mu  sync.Mutex
	r   io.Reader
	buf []byte
	tmp [constants.MJPEGScannerChunk]byte
}

// NewScanner создаёт сканер поверх произвольного io.Reader.
func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: r}
}

// ReadFrameCtx читает следующий кадр; отменяется при отмене ctx (shutdown, переподключение).
func (s *Scanner) ReadFrameCtx(ctx context.Context) ([]byte, error) {
	type result struct {
		b   []byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		b, err := s.ReadFrame()
		ch <- result{b, err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.b, r.err
	}
}

// ReadFrame возвращает следующий JPEG (SOI … EOI) или ошибку.
func (s *Scanner) ReadFrame() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	soi := []byte{0xff, 0xd8}
	eoi := []byte{0xff, 0xd9}
	maxJPEG := constants.MaxJPEGSize

	for len(s.buf) < maxJPEG {
		idx := bytes.Index(s.buf, soi)
		if idx < 0 {
			if len(s.buf) > 65536 {
				s.buf = s.buf[len(s.buf)-1:]
			}
			n, err := s.r.Read(s.tmp[:])
			s.buf = append(s.buf, s.tmp[:n]...)
			if err != nil {
				if err == io.EOF && n == 0 {
					return nil, io.EOF
				}
				if err != io.EOF {
					return nil, err
				}
			}
			continue
		}
		if idx > 0 {
			s.buf = s.buf[idx:]
		}
		endRel := bytes.Index(s.buf[2:], eoi)
		if endRel < 0 {
			n, err := s.r.Read(s.tmp[:])
			s.buf = append(s.buf, s.tmp[:n]...)
			if err != nil {
				if err == io.EOF {
					return nil, io.ErrUnexpectedEOF
				}
				return nil, err
			}
			continue
		}
		frameLen := 2 + endRel + 2
		frame := make([]byte, frameLen)
		copy(frame, s.buf[:frameLen])
		s.buf = s.buf[frameLen:]
		return frame, nil
	}
	return nil, io.ErrUnexpectedEOF
}
