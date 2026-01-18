package lsp

import (
	"strings"
	"sync"
)

type Buffer struct {
	URI        string
	Text       string
	Version    int
	LanguageID string
}

type BufferStore struct {
	mu         sync.RWMutex
	buffers    map[string]*Buffer
	currentURI string
}

func NewBufferStore() *BufferStore {
	return &BufferStore{
		buffers: make(map[string]*Buffer),
	}
}

func (s *BufferStore) Set(buf *Buffer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buffers[buf.URI] = buf
	s.currentURI = buf.URI
}

func (s *BufferStore) Get(uri string) (*Buffer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	buf, ok := s.buffers[uri]
	return buf, ok
}

func (s *BufferStore) GetCurrent() (*Buffer, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentURI == "" {
		return nil, false
	}
	buf, ok := s.buffers[s.currentURI]
	return buf, ok
}

func (s *BufferStore) CurrentURI() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentURI
}

func (s *BufferStore) SetCurrentURI(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentURI = uri
}

func (s *BufferStore) UpdateText(uri string, version int, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if buf, ok := s.buffers[uri]; ok {
		buf.Text = text
		buf.Version = version
	}
	s.currentURI = uri
}

func (s *BufferStore) Delete(uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buffers, uri)
}

func (s *BufferStore) GetContentFromRange(uri string, r Range) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	buf, ok := s.buffers[uri]

	if !ok || buf.Text == "" {
		return ""
	}

	lines := strings.Split(buf.Text, "\n")

	if r.Start.Line >= len(lines) {
		return ""
	}

	endLine := r.End.Line

	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	selectedLines := lines[r.Start.Line : endLine+1]
	return strings.Join(selectedLines, "\n")
}
