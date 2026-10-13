package umabridge

import (
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	msgtypes "hop.top/aps/internal/core/messenger"
	"hop.top/uma/model"
)

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// materializeMedia copies the attachments staged for the turn's message out
// of the engine into dir/inbound/<message>/<n><ext>, owner-only, and returns
// them as aps attachments with file:// URLs: aps actions run on the same
// host, so a local file is the one form every action can read. A message
// without attachments yields none.
func materializeMedia(ctx context.Context, st *turnState, dir, messageID string) ([]msgtypes.Attachment, error) {
	var out []msgtypes.Attachment
	target := filepath.Join(dir, "inbound", unsafeName.ReplaceAllString(messageID, "_"))
	n := 0
	for _, part := range st.turn.Message.Content.Parts {
		if part.Kind != model.PartAttachment || part.Attachment == nil {
			continue
		}
		att, body, err := st.client.Attachments().Open(ctx, model.Lookup[model.AttachmentRef]{
			Call: model.Call{Correlation: "aps-bridge-media"}, Ref: *part.Attachment,
		})
		if err != nil {
			return nil, fmt.Errorf("open attachment: %w", err)
		}
		path, size, err := writeMedia(target, n, att.Metadata.MediaType, body)
		_ = body.Close()
		if err != nil {
			return nil, err
		}
		out = append(out, msgtypes.Attachment{
			Type: attachmentType(att.Metadata.MediaType), URL: "file://" + path,
			MimeType: att.Metadata.MediaType, SizeBytes: size,
		})
		n++
	}
	return out, nil
}

func writeMedia(dir string, n int, mediaType string, r io.Reader) (string, int64, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", 0, fmt.Errorf("create media dir: %w", err)
	}
	ext := ""
	if exts, _ := mime.ExtensionsByType(mediaType); len(exts) > 0 {
		ext = exts[0]
	}
	path := filepath.Join(dir, strconv.Itoa(n)+ext)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- under the configured media dir, name built here
	if err != nil {
		return "", 0, fmt.Errorf("create media file: %w", err)
	}
	size, copyErr := io.Copy(f, r)
	if closeErr := f.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(path)
		return "", 0, fmt.Errorf("write media file: %w", copyErr)
	}
	return path, size, nil
}

func attachmentType(mediaType string) string {
	switch {
	case strings.HasPrefix(mediaType, "image/"):
		return "image"
	case strings.HasPrefix(mediaType, "audio/"):
		return "audio"
	case strings.HasPrefix(mediaType, "video/"):
		return "video"
	}
	return "file"
}
