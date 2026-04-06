package dory

import (
	"crypto/sha256"
	"fmt"
	"io"
	"time"
)

// Document is the ingestion unit — a raw source of knowledge before
// it has been chunked or indexed. A document carries its content,
// its identity, and the metadata the authorizer and retriever will
// consult later.
//
// Documents are created via [NewDocument], which validates required
// fields and computes a content fingerprint for change detection.
type Document struct {
	id        string
	content   Content
	tenantID  string
	sourceURI string
	language  string
	metadata  map[string]any
	createdAt time.Time
	updatedAt time.Time

	// fingerprint is a SHA-256 hash of the content bytes,
	// computed at construction time. Used to detect whether
	// a document has changed since its last ingestion.
	fingerprint [32]byte
}

// DocumentOption configures a Document at construction time.
type DocumentOption func(*Document) error

// NewDocument constructs a validated Document.
// Returns an error if the document cannot be used by Dory's pipeline —
// for example, if the ID is empty or the content is nil.
func NewDocument(id string, content Content, opts ...DocumentOption) (*Document, error) {
	if id == "" {
		return nil, fmt.Errorf("dory: document ID must not be empty")
	}
	if content == nil {
		return nil, fmt.Errorf("dory: document content must not be nil")
	}

	now := time.Now().UTC()
	doc := &Document{
		id:        id,
		content:   content,
		language:  "en",
		metadata:  make(map[string]any),
		createdAt: now,
		updatedAt: now,
	}

	// Compute content fingerprint.
	rc, err := content.Reader()
	if err != nil {
		return nil, fmt.Errorf("dory: cannot read content for fingerprinting: %w", err)
	}
	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		_ = rc.Close()
		return nil, fmt.Errorf("dory: cannot hash content: %w", err)
	}
	_ = rc.Close()
	copy(doc.fingerprint[:], h.Sum(nil))

	for _, opt := range opts {
		if err := opt(doc); err != nil {
			return nil, fmt.Errorf("dory: document option: %w", err)
		}
	}

	return doc, nil
}

// ID returns the document's unique identifier.
func (d *Document) ID() string { return d.id }

// Content returns the document's content.
func (d *Document) Content() Content { return d.content }

// TenantID returns the tenant this document belongs to.
func (d *Document) TenantID() string { return d.tenantID }

// SourceURI returns the canonical location of this document's original source.
func (d *Document) SourceURI() string { return d.sourceURI }

// Language returns the BCP-47 language tag for this document's content.
func (d *Document) Language() string { return d.language }

// Metadata returns the document's metadata.
func (d *Document) Metadata() map[string]any { return d.metadata }

// CreatedAt returns when this document was first ingested.
func (d *Document) CreatedAt() time.Time { return d.createdAt }

// UpdatedAt returns the last time this document was re-ingested.
func (d *Document) UpdatedAt() time.Time { return d.updatedAt }

// Fingerprint returns the SHA-256 hash of this document's content.
// If two Documents have the same ID and the same Fingerprint,
// re-ingestion can be skipped safely.
func (d *Document) Fingerprint() [32]byte { return d.fingerprint }

// --- Document Options ---

// WithTenantID sets the tenant this document belongs to.
func WithTenantID(id string) DocumentOption {
	return func(d *Document) error {
		if id == "" {
			return fmt.Errorf("tenant ID must not be empty")
		}
		d.tenantID = id
		return nil
	}
}

// WithSourceURI sets the canonical source location for this document.
// Examples: "s3://bucket/path/to/file.pdf", "https://docs.example.com/api".
func WithSourceURI(uri string) DocumentOption {
	return func(d *Document) error {
		d.sourceURI = uri
		return nil
	}
}

// WithLanguage sets the BCP-47 language tag for this document's content.
// Used by sentence-aware chunking strategies to apply the correct
// sentence boundary detection rules. Defaults to "en" if not set.
func WithLanguage(tag string) DocumentOption {
	return func(d *Document) error {
		if tag == "" {
			return fmt.Errorf("language tag must not be empty")
		}
		d.language = tag
		return nil
	}
}

// WithMetadata attaches a key-value pair to the document's metadata.
func WithMetadata(key string, value any) DocumentOption {
	return func(d *Document) error {
		d.metadata[key] = value
		return nil
	}
}

// WithTimestamps overrides the default creation and update timestamps.
func WithTimestamps(createdAt, updatedAt time.Time) DocumentOption {
	return func(d *Document) error {
		d.createdAt = createdAt
		d.updatedAt = updatedAt
		return nil
	}
}
