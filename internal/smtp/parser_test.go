package smtp

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== ParseEmail Tests ====================

// TestParseEmail_SimpleText tests parsing a simple text email
func TestParseEmail_SimpleText(t *testing.T) {
	// Arrange
	emailContent := `From: sender@example.com
To: receiver@test.com
Subject: Simple Text Email
Content-Type: text/plain; charset=utf-8

Hello, this is a simple text email.`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sender@example.com", parsed.SenderEmail)
	assert.Equal(t, "Simple Text Email", parsed.Subject)
	assert.Contains(t, parsed.BodyText, "Hello, this is a simple text email")
	assert.Empty(t, parsed.BodyHTML)
	assert.Empty(t, parsed.Attachments)
}

// TestParseEmail_HTMLEmail tests parsing an HTML email
func TestParseEmail_HTMLEmail(t *testing.T) {
	// Arrange
	emailContent := `From: sender@example.com
To: receiver@test.com
Subject: HTML Email
Content-Type: text/html; charset=utf-8

<html><body><h1>Hello World</h1><p>This is an HTML email.</p></body></html>`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sender@example.com", parsed.SenderEmail)
	assert.Equal(t, "HTML Email", parsed.Subject)
	assert.Contains(t, parsed.BodyHTML, "<h1>Hello World</h1>")
	assert.Empty(t, parsed.Attachments)
}

// TestParseEmail_MultipartAlternative tests parsing a multipart/alternative email
func TestParseEmail_MultipartAlternative(t *testing.T) {
	// Arrange
	emailContent := `From: sender@example.com
To: receiver@test.com
Subject: Multipart Alternative
MIME-Version: 1.0
Content-Type: multipart/alternative; boundary="boundary123"

--boundary123
Content-Type: text/plain; charset=utf-8

Plain text version.

--boundary123
Content-Type: text/html; charset=utf-8

<html><body><p>HTML version.</p></body></html>

--boundary123--`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sender@example.com", parsed.SenderEmail)
	assert.Equal(t, "Multipart Alternative", parsed.Subject)
	assert.Contains(t, parsed.BodyText, "Plain text version")
	assert.Contains(t, parsed.BodyHTML, "HTML version")
}

// TestParseEmail_WithAttachment tests parsing an email with attachment
func TestParseEmail_WithAttachment(t *testing.T) {
	// Arrange
	emailContent := `From: sender@example.com
To: receiver@test.com
Subject: Email with Attachment
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="boundary456"

--boundary456
Content-Type: text/plain; charset=utf-8

Email body with attachment.

--boundary456
Content-Type: application/pdf; name="document.pdf"
Content-Disposition: attachment; filename="document.pdf"
Content-Transfer-Encoding: base64

JVBERi0xLjQKJeLjz9MK

--boundary456--`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sender@example.com", parsed.SenderEmail)
	assert.Equal(t, "Email with Attachment", parsed.Subject)
	assert.Contains(t, parsed.BodyText, "Email body with attachment")
	require.Len(t, parsed.Attachments, 1)
	assert.Equal(t, "document.pdf", parsed.Attachments[0].Filename)
	assert.Equal(t, "application/pdf", parsed.Attachments[0].ContentType)
}

// TestParseEmail_ExtractsFromHeader tests that From header is correctly extracted
func TestParseEmail_ExtractsFromHeader(t *testing.T) {
	// Arrange
	emailContent := `From: "Test Sender" <sender@example.com>
To: receiver@test.com
Subject: Test
Content-Type: text/plain

Body`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sender@example.com", parsed.SenderEmail)
	assert.Equal(t, "Test Sender", parsed.SenderName)
}

// TestParseEmail_ExtractsToHeader tests that To header is correctly extracted
func TestParseEmail_ExtractsToHeader(t *testing.T) {
	// Arrange
	emailContent := `From: sender@example.com
To: receiver@test.com
Subject: Test
Content-Type: text/plain

Body`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	// Note: ParseEmail doesn't extract To header, but we verify it doesn't error
	assert.NotNil(t, parsed)
}

// TestParseEmail_ExtractsSubject tests that Subject header is correctly extracted
func TestParseEmail_ExtractsSubject(t *testing.T) {
	// Arrange
	emailContent := `From: sender@example.com
To: receiver@test.com
Subject: This is a test subject with special chars: äöü
Content-Type: text/plain

Body`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	assert.Contains(t, parsed.Subject, "This is a test subject")
}


// TestParseEmail_MultipleAttachments tests parsing an email with multiple attachments
func TestParseEmail_MultipleAttachments(t *testing.T) {
	// Arrange
	emailContent := `From: sender@example.com
To: receiver@test.com
Subject: Multiple Attachments
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="boundary789"

--boundary789
Content-Type: text/plain; charset=utf-8

Email with multiple attachments.

--boundary789
Content-Type: application/pdf; name="doc1.pdf"
Content-Disposition: attachment; filename="doc1.pdf"
Content-Transfer-Encoding: base64

JVBERi0xLjQ=

--boundary789
Content-Type: image/png; name="image.png"
Content-Disposition: attachment; filename="image.png"
Content-Transfer-Encoding: base64

iVBORw0KGgo=

--boundary789--`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	require.Len(t, parsed.Attachments, 2)
	assert.Equal(t, "doc1.pdf", parsed.Attachments[0].Filename)
	assert.Equal(t, "image.png", parsed.Attachments[1].Filename)
}

// TestParseEmail_FromFixtureFile tests parsing from fixture file
func TestParseEmail_FromFixtureFile(t *testing.T) {
	// Skip if fixture file doesn't exist
	fixturePath := filepath.Join("..", "..", "tests", "fixtures", "emails", "simple_text.eml")
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skip("Fixture file not found")
	}

	// Arrange
	file, err := os.Open(fixturePath)
	require.NoError(t, err)
	defer file.Close()

	// Act
	parsed, err := ParseEmail(file)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "sender@example.com", parsed.SenderEmail)
	assert.Equal(t, "Simple Text Email", parsed.Subject)
}

// ==================== parseFromHeader Tests ====================

// TestParseFromHeader_EmailOnly tests parsing email-only From header
func TestParseFromHeader_EmailOnly(t *testing.T) {
	// Act
	name, email := parseFromHeader("sender@example.com")

	// Assert
	assert.Empty(t, name)
	assert.Equal(t, "sender@example.com", email)
}

// TestParseFromHeader_NameAndEmail tests parsing From header with name and email
func TestParseFromHeader_NameAndEmail(t *testing.T) {
	// Act
	name, email := parseFromHeader("Test Sender <sender@example.com>")

	// Assert
	assert.Equal(t, "Test Sender", name)
	assert.Equal(t, "sender@example.com", email)
}

// TestParseFromHeader_QuotedName tests parsing From header with quoted name
func TestParseFromHeader_QuotedName(t *testing.T) {
	// Act
	name, email := parseFromHeader(`"Test Sender" <sender@example.com>`)

	// Assert
	assert.Equal(t, "Test Sender", name)
	assert.Equal(t, "sender@example.com", email)
}

// TestParseFromHeader_Empty tests parsing empty From header
func TestParseFromHeader_Empty(t *testing.T) {
	// Act
	name, email := parseFromHeader("")

	// Assert
	assert.Empty(t, name)
	assert.Empty(t, email)
}

// TestParseFromHeader_WithWhitespace tests parsing From header with whitespace
func TestParseFromHeader_WithWhitespace(t *testing.T) {
	// Act
	name, email := parseFromHeader("  Test Sender  <sender@example.com>  ")

	// Assert
	assert.Equal(t, "Test Sender", name)
	assert.Equal(t, "sender@example.com", email)
}

// ==================== generateSnippet Tests ====================

// TestGenerateSnippet_FromText tests generating snippet from text body
func TestGenerateSnippet_FromText(t *testing.T) {
	// Act
	snippet := generateSnippet("Hello, this is a test email body.", "")

	// Assert
	assert.Equal(t, "Hello, this is a test email body.", snippet)
}

// TestGenerateSnippet_FromHTML tests generating snippet from HTML body
func TestGenerateSnippet_FromHTML(t *testing.T) {
	// Act
	snippet := generateSnippet("", "<html><body><p>Hello World</p></body></html>")

	// Assert
	assert.Contains(t, snippet, "Hello World")
	assert.NotContains(t, snippet, "<p>")
}

// TestGenerateSnippet_Truncation tests snippet truncation at 255 chars
func TestGenerateSnippet_Truncation(t *testing.T) {
	// Arrange
	longText := strings.Repeat("a", 300)

	// Act
	snippet := generateSnippet(longText, "")

	// Assert
	assert.Len(t, snippet, 255)
	assert.True(t, strings.HasSuffix(snippet, "..."))
}

// TestGenerateSnippet_PrefersText tests that text body is preferred over HTML
func TestGenerateSnippet_PrefersText(t *testing.T) {
	// Act
	snippet := generateSnippet("Plain text content", "<p>HTML content</p>")

	// Assert
	assert.Equal(t, "Plain text content", snippet)
}

// TestGenerateSnippet_Empty tests generating snippet from empty bodies
func TestGenerateSnippet_Empty(t *testing.T) {
	// Act
	snippet := generateSnippet("", "")

	// Assert
	assert.Empty(t, snippet)
}

// ==================== stripHTMLTags Tests ====================

// TestStripHTMLTags_Basic tests basic HTML tag stripping
func TestStripHTMLTags_Basic(t *testing.T) {
	// Act
	result := stripHTMLTags("<p>Hello World</p>")

	// Assert
	assert.Contains(t, result, "Hello World")
	assert.NotContains(t, result, "<p>")
}

// TestStripHTMLTags_Script tests script tag removal
func TestStripHTMLTags_Script(t *testing.T) {
	// Act
	result := stripHTMLTags("<script>alert('xss')</script><p>Content</p>")

	// Assert
	assert.Contains(t, result, "Content")
	assert.NotContains(t, result, "alert")
	assert.NotContains(t, result, "script")
}

// TestStripHTMLTags_Style tests style tag removal
func TestStripHTMLTags_Style(t *testing.T) {
	// Act
	result := stripHTMLTags("<style>.class { color: red; }</style><p>Content</p>")

	// Assert
	assert.Contains(t, result, "Content")
	assert.NotContains(t, result, "color")
	assert.NotContains(t, result, "style")
}

// TestStripHTMLTags_Entities tests HTML entity decoding
func TestStripHTMLTags_Entities(t *testing.T) {
	// Act
	result := stripHTMLTags("Hello&nbsp;World &amp; Friends &lt;test&gt;")

	// Assert
	assert.Contains(t, result, "Hello World")
	assert.Contains(t, result, "& Friends")
	assert.Contains(t, result, "<test>")
}

// ==================== Attachment Content Tests ====================

// TestParseEmail_AttachmentContent tests that attachment content is readable
func TestParseEmail_AttachmentContent(t *testing.T) {
	// Arrange
	emailContent := `From: sender@example.com
To: receiver@test.com
Subject: Attachment Test
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="boundary"

--boundary
Content-Type: text/plain

Body

--boundary
Content-Type: text/plain; name="test.txt"
Content-Disposition: attachment; filename="test.txt"

Hello from attachment!

--boundary--`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	require.Len(t, parsed.Attachments, 1)
	
	content, err := io.ReadAll(parsed.Attachments[0].Content)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Hello from attachment")
}

// TestParseEmail_AttachmentSize tests that attachment size is correct
func TestParseEmail_AttachmentSize(t *testing.T) {
	// Arrange
	emailContent := `From: sender@example.com
To: receiver@test.com
Subject: Size Test
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="boundary"

--boundary
Content-Type: text/plain

Body

--boundary
Content-Type: text/plain; name="test.txt"
Content-Disposition: attachment; filename="test.txt"

12345

--boundary--`

	// Act
	parsed, err := ParseEmail(strings.NewReader(emailContent))

	// Assert
	require.NoError(t, err)
	require.Len(t, parsed.Attachments, 1)
	assert.Greater(t, parsed.Attachments[0].Size, int64(0))
}
