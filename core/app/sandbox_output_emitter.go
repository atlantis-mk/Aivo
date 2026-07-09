package app

import (
	"bytes"
	"io"
	"time"

	"aivo/core/domain"
)

func newShellOutputEmitter(request SandboxRequest, processRef string) *shellOutputEmitter {
	if request.OutputSink == nil {
		return nil
	}
	return &shellOutputEmitter{request: request, processRef: processRef}
}

func newShellOutputWriter(target *bytes.Buffer, emitter *shellOutputEmitter, stream string) io.Writer {
	if emitter == nil {
		return target
	}
	return &shellOutputWriter{target: target, emitter: emitter, stream: stream}
}

func (w *shellOutputWriter) Write(p []byte) (int, error) {
	n, err := w.target.Write(p)
	if n > 0 {
		w.emitter.emit(w.stream, string(p[:n]))
	}
	return n, err
}

func (e *shellOutputEmitter) emit(stream string, chunk string) {
	if e == nil || e.request.OutputSink == nil || chunk == "" {
		return
	}
	e.mu.Lock()
	e.nextSeq++
	sequence := e.nextSeq
	e.mu.Unlock()
	e.request.OutputSink(ShellOutputEvent{
		SessionID:   e.request.SessionID,
		TurnID:      e.request.TurnID,
		ToolCallID:  e.request.ToolCallID,
		ProcessRef:  e.processRef,
		Stream:      stream,
		Chunk:       redactCommandOutput(chunk),
		Sequence:    sequence,
		TimeCreated: domain.NowString(time.Now()),
	})
}
