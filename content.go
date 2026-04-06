package dory

import (
	"bytes"
	"io"
)

// Content is the raw material of a Document.
// It abstracts over text, binary, and streaming content
// so that Dory's pipeline can handle each appropriately.
type Content interface {
	// Reader returns the content as a stream of bytes.
	// Callers are responsible for closing the reader.
	Reader() (io.ReadCloser, error)

	// Text returns the content as a UTF-8 string, if possible.
	// Returns an error if the content is binary or not yet extracted.
	// Splitters call this — they work on text, not bytes.
	Text() (string, error)

	// MimeType describes the format of the content.
	MimeType() string

	// Size returns the content length in bytes, or -1 if unknown.
	Size() int64
}

// StringContent is a Content backed by a plain UTF-8 string.
// This is the most common case for pre-extracted text.
type StringContent struct {
	text     string
	mimeType string
}

// TextContent creates a StringContent with the given text and mime type.
// Pass an empty mimeType to default to "text/plain".
func TextContent(text, mimeType string) *StringContent {
	if mimeType == "" {
		mimeType = "text/plain"
	}
	return &StringContent{text: text, mimeType: mimeType}
}

func (s *StringContent) Reader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte(s.text))), nil
}

func (s *StringContent) Text() (string, error) {
	return s.text, nil
}

func (s *StringContent) MimeType() string {
	return s.mimeType
}

func (s *StringContent) Size() int64 {
	return int64(len(s.text))
}

// BytesContent is a Content backed by raw bytes.
// Use this for binary formats like PDF or images where the
// content has not yet been extracted to text.
type BytesContent struct {
	data     []byte
	mimeType string
}

// BinaryContent creates a BytesContent with the given data and mime type.
func BinaryContent(data []byte, mimeType string) *BytesContent {
	return &BytesContent{data: data, mimeType: mimeType}
}

func (b *BytesContent) Reader() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.data)), nil
}

func (b *BytesContent) Text() (string, error) {
	return string(b.data), nil
}

func (b *BytesContent) MimeType() string {
	return b.mimeType
}

func (b *BytesContent) Size() int64 {
	return int64(len(b.data))
}

// ReaderContent is a Content backed by a lazy reader function.
// Use this for streaming large files without loading them into memory.
type ReaderContent struct {
	open     func() (io.ReadCloser, error)
	mimeType string
	size     int64
}

// StreamContent creates a ReaderContent with the given reader factory.
// The open function is called each time Reader() is invoked, allowing
// multiple reads of the same content. Pass size=-1 if the size is unknown.
func StreamContent(open func() (io.ReadCloser, error), mimeType string, size int64) *ReaderContent {
	return &ReaderContent{open: open, mimeType: mimeType, size: size}
}

func (r *ReaderContent) Reader() (io.ReadCloser, error) {
	return r.open()
}

func (r *ReaderContent) Text() (string, error) {
	rc, err := r.open()
	if err != nil {
		return "", err
	}
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *ReaderContent) MimeType() string {
	return r.mimeType
}

func (r *ReaderContent) Size() int64 {
	return r.size
}
