package generation

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

const progressBarWidth = 28

var progressSessionActive bool
var progressSessionRows int

type generationProgress struct {
	mu             sync.Mutex
	interactive    bool
	started        bool
	finished       bool
	name           string
	version        string
	showDiscovery  bool
	showReading    bool
	showDocs       bool
	showFinalizing bool
	showOG         bool
	rows           int
	discovery      bool
	finalizing     bool
	reading        progressState
	documenting    progressState
	og             progressState
}

type progressState struct {
	label    string
	current  int
	total    int
	message  string
	deferred bool
}

type progressOptions struct {
	Discovery     bool
	Reading       bool
	Documentation bool
	Finalizing    bool
	OG            bool
}

func newGenerationProgress(name, version string, options progressOptions, quiet ...bool) *generationProgress {
	interactive := true
	if len(quiet) > 0 {
		interactive = !quiet[0]
	}
	progress := &generationProgress{
		interactive:    interactive,
		name:           name,
		version:        version,
		showDiscovery:  options.Discovery,
		showReading:    options.Reading,
		showDocs:       options.Documentation,
		showFinalizing: options.Finalizing,
		showOG:         options.OG,
		reading:        progressState{label: "📄 Reading WGSL Files"},
		documenting:    progressState{label: "🛠️ Generating Documentation"},
		og:             progressState{label: "🖼 Generation OG image"},
	}
	progress.rows = 1
	if options.Discovery {
		progress.rows++
	}
	if options.Reading {
		progress.rows++
	}
	if options.Documentation {
		progress.rows++
	}
	if options.Finalizing {
		progress.rows++
	}
	if options.OG {
		progress.rows++
	}
	if progress.interactive && progressSessionActive {
		fmt.Fprintf(os.Stdout, "\033[%dA", progressSessionRows)
	}
	progress.render()
	if progress.interactive {
		progressSessionActive = true
		progressSessionRows = progress.rows
	}
	return progress
}

func (p *generationProgress) completeDiscovery() {
	p.mu.Lock()
	p.discovery = true
	p.mu.Unlock()
	p.render()
}

func (p *generationProgress) completeFinalizing() {
	p.mu.Lock()
	p.finalizing = true
	p.mu.Unlock()
	p.render()
}

func (p *generationProgress) setReadingTotal(total int) {
	p.mu.Lock()
	p.reading.total = total
	p.mu.Unlock()
	p.render()
}

func (p *generationProgress) setDocumentationTotal(total int) {
	p.mu.Lock()
	p.documenting.total = total
	p.mu.Unlock()
	p.render()
}

func (p *generationProgress) setOGTotal(total int) {
	if !p.showOG {
		return
	}
	p.mu.Lock()
	p.og.total = total
	p.og.deferred = false
	p.mu.Unlock()
	p.render()
}

func (p *generationProgress) addReading()       { p.add(&p.reading) }
func (p *generationProgress) addDocumentation() { p.add(&p.documenting) }
func (p *generationProgress) addOG()            { p.add(&p.og) }

func (p *generationProgress) add(state *progressState) {
	p.mu.Lock()
	if state.total == 0 || state.current < state.total {
		state.current++
	}
	p.mu.Unlock()
	p.render()
}

func (p *generationProgress) finish() {
	p.mu.Lock()
	if p.finished {
		p.mu.Unlock()
		return
	}
	p.finished = true
	p.mu.Unlock()
	p.render()
}

func (p *generationProgress) render() {
	p.mu.Lock()
	defer p.mu.Unlock()
	lines := []string{fmt.Sprintf("🚀 Generating documentation for %s %s", p.name, p.version)}
	if p.showDiscovery {
		lines = append(lines, formatStatus("🔎 Discovering packages and dependencies", p.discovery))
	}
	if p.showReading {
		lines = append(lines, formatProgress(p.reading))
	}
	if p.showDocs {
		lines = append(lines, formatProgress(p.documenting))
	}
	if p.showFinalizing {
		lines = append(lines, formatStatus("✅ Finalizing catalogue", p.finalizing))
	}
	if p.showOG {
		lines = append(lines, formatProgress(p.og))
	}
	if p.interactive && p.started {
		fmt.Fprintf(os.Stdout, "\033[%dA", p.rows)
	}
	if p.interactive {
		for _, line := range lines {
			fmt.Fprintf(os.Stdout, "\r\033[2K%s\n", line)
		}
	}
	p.started = true
}

func formatStatus(label string, complete bool) string {
	if complete {
		return label + " — complete"
	}
	return label + " ..."
}

func formatProgress(state progressState) string {
	if state.deferred {
		return fmt.Sprintf("%s — %s", state.label, state.message)
	}
	if state.total <= 0 {
		return state.label + " — waiting"
	}
	percent := state.current * 100 / state.total
	if percent > 100 {
		percent = 100
	}
	filled := state.current * progressBarWidth / state.total
	if filled > progressBarWidth {
		filled = progressBarWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)
	return fmt.Sprintf("%s %3d%% [%s] (%d/%d)", state.label, percent, bar, state.current, state.total)
}
