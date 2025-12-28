package smtp

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/jhillyerd/enmime"
)

// ParsedEmail represents a parsed email message
type ParsedEmail struct {
	SenderEmail string
	SenderName  string
	Subject     string
	Snippet     string
	BodyText    string
	BodyHTML    string
	Attachments []ParsedAttachment
}

// ParsedAttachment represents a parsed email attachment
type ParsedAttachment struct {
	Filename    string
	ContentType string
	Content     io.Reader
	Size        int64
}

// ParseEmail parses an email from an io.Reader
func ParseEmail(r io.Reader) (*ParsedEmail, error) {
	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return nil, err
	}

	parsed := &ParsedEmail{
		Subject:  env.GetHeader("Subject"),
		BodyText: env.Text,
		BodyHTML: env.HTML,
	}

	// Parse From header
	fromHeader := env.GetHeader("From")
	parsed.SenderName, parsed.SenderEmail = parseFromHeader(fromHeader)

	// Generate snippet
	parsed.Snippet = generateSnippet(parsed.BodyText, parsed.BodyHTML)

	// Parse attachments
	for _, att := range env.Attachments {
		parsed.Attachments = append(parsed.Attachments, ParsedAttachment{
			Filename:    att.FileName,
			ContentType: att.ContentType,
			Content:     bytes.NewReader(att.Content),
			Size:        int64(len(att.Content)),
		})
	}

	// Also include inline attachments
	for _, att := range env.Inlines {
		if att.FileName != "" {
			parsed.Attachments = append(parsed.Attachments, ParsedAttachment{
				Filename:    att.FileName,
				ContentType: att.ContentType,
				Content:     bytes.NewReader(att.Content),
				Size:        int64(len(att.Content)),
			})
		}
	}

	return parsed, nil
}

// parseFromHeader extracts name and email from a From header
func parseFromHeader(from string) (name, email string) {
	from = strings.TrimSpace(from)
	if from == "" {
		return "", ""
	}

	// Pattern: "Name" <email@example.com> or Name <email@example.com>
	re := regexp.MustCompile(`^(?:"?([^"<]*)"?\s*)?<?([^<>]+@[^<>]+)>?$`)
	matches := re.FindStringSubmatch(from)

	if len(matches) >= 3 {
		name = strings.TrimSpace(matches[1])
		email = strings.TrimSpace(matches[2])
		// Remove quotes from name
		name = strings.Trim(name, `"`)
	} else {
		// Fallback: treat entire string as email
		email = from
	}

	return name, email
}

// generateSnippet creates a preview snippet from email body
func generateSnippet(bodyText, bodyHTML string) string {
	var text string

	if bodyText != "" {
		text = bodyText
	} else if bodyHTML != "" {
		// Strip HTML tags
		text = stripHTMLTags(bodyHTML)
	}

	// Clean up whitespace
	text = strings.Join(strings.Fields(text), " ")
	text = strings.TrimSpace(text)

	// Truncate to 255 characters
	if len(text) > 255 {
		text = text[:252] + "..."
	}

	return text
}

// stripHTMLTags removes HTML tags from a string
func stripHTMLTags(html string) string {
	// Remove script and style elements
	re := regexp.MustCompile(`(?i)<(script|style)[^>]*>[\s\S]*?</\1>`)
	html = re.ReplaceAllString(html, "")

	// Remove HTML tags
	re = regexp.MustCompile(`<[^>]*>`)
	html = re.ReplaceAllString(html, " ")

	// Decode common HTML entities
	html = strings.ReplaceAll(html, "&nbsp;", " ")
	html = strings.ReplaceAll(html, "&amp;", "&")
	html = strings.ReplaceAll(html, "&lt;", "<")
	html = strings.ReplaceAll(html, "&gt;", ">")
	html = strings.ReplaceAll(html, "&quot;", `"`)
	html = strings.ReplaceAll(html, "&#39;", "'")

	return html
}
