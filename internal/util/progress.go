package util

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/leona/helix-assist/internal/config"
	"github.com/leona/helix-assist/internal/lsp"
)

type ProgressIndicator struct {
	svc            *lsp.Service
	enabled        bool
	updateInterval time.Duration
	spinnerFrames  []string
	ctx            context.Context
	cancel         context.CancelFunc
	startTime      time.Time
	diagRange      lsp.Range
	timeout        int
	mu             sync.Mutex
}

func NewProgressIndicator(svc *lsp.Service, cfg *config.Config, diagRange lsp.Range, timeout int) *ProgressIndicator {
	return &ProgressIndicator{
		svc:            svc,
		enabled:        cfg.EnableProgressSpinner,
		updateInterval: time.Duration(cfg.ProgressUpdateInterval) * time.Millisecond,
		spinnerFrames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		diagRange:      diagRange,
		timeout:        timeout,
	}
}

func (p *ProgressIndicator) Start() {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.startTime = time.Now()
	p.mu.Unlock()

	go p.animate()
}

func (p *ProgressIndicator) Stop() {
	if !p.enabled {
		return
	}

	p.mu.Lock()

	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}

	p.mu.Unlock()
	p.svc.ResetDiagnostics()
}

func (p *ProgressIndicator) animate() {
	ticker := time.NewTicker(p.updateInterval)
	defer ticker.Stop()

	frameIndex := 0

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(p.startTime)
			message := fmt.Sprintf("%s (%s)", p.spinnerFrames[frameIndex], p.formatElapsed(elapsed))

			p.svc.SendDiagnostics([]lsp.Diagnostic{
				{
					Message:  message,
					Severity: lsp.SeverityInformation,
					Range:    p.diagRange,
				},
			}, p.timeout)

			frameIndex = (frameIndex + 1) % len(p.spinnerFrames)
		}
	}
}

func (p *ProgressIndicator) formatElapsed(duration time.Duration) string {
	seconds := duration.Seconds()
	return fmt.Sprintf("%.1fs", seconds)
}
