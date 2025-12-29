package services

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestDNSGuide() *DNSGuide {
	return &DNSGuide{
		SMTPHost: "mail.infinimail.webrana.id",
		ServerIP: "103.123.45.67",
		MXRecord: DNSRecord{
			Type:     "MX",
			Name:     "example.com",
			Value:    "mail.infinimail.webrana.id",
			Priority: 10,
			TTL:      3600,
		},
		ARecord: DNSRecord{
			Type:  "A",
			Name:  "mail.example.com",
			Value: "103.123.45.67",
			TTL:   3600,
		},
		TXTRecord: DNSRecord{
			Type:  "TXT",
			Name:  "_infinimail.example.com",
			Value: "infinimail-verify=abc123xyz",
			TTL:   3600,
		},
	}
}

func TestExportBIND(t *testing.T) {
	exporter := NewDNSExporter()
	guide := createTestDNSGuide()

	result := exporter.ExportBIND(guide, "example.com")

	// Cloudflare-compatible BIND format (no $TTL directive, TTL per record, tab-separated)
	// Verify MX record with @ for root domain
	assert.Contains(t, result, "@\t3600\tIN\tMX\t10\tmail.infinimail.webrana.id.")

	// Verify A record with subdomain extracted
	assert.Contains(t, result, "mail\t3600\tIN\tA\t103.123.45.67")

	// Verify TXT record with subdomain extracted
	assert.Contains(t, result, "_infinimail\t3600\tIN\tTXT\t\"infinimail-verify=abc123xyz\"")
}

func TestExportCloudflare(t *testing.T) {
	exporter := NewDNSExporter()
	guide := createTestDNSGuide()

	records, err := exporter.ExportCloudflare(guide, "example.com")

	assert.NoError(t, err)
	assert.Len(t, records, 3)

	// Verify MX record
	mxRecord := records[0]
	assert.Equal(t, "MX", mxRecord.Type)
	assert.Equal(t, "@", mxRecord.Name)
	assert.Equal(t, "mail.infinimail.webrana.id", mxRecord.Content)
	assert.Equal(t, 10, mxRecord.Priority)
	assert.Equal(t, 3600, mxRecord.TTL)
	assert.False(t, mxRecord.Proxied)

	// Verify A record
	aRecord := records[1]
	assert.Equal(t, "A", aRecord.Type)
	assert.Equal(t, "mail", aRecord.Name)
	assert.Equal(t, "103.123.45.67", aRecord.Content)
	assert.Equal(t, 3600, aRecord.TTL)
	assert.False(t, aRecord.Proxied)

	// Verify TXT record
	txtRecord := records[2]
	assert.Equal(t, "TXT", txtRecord.Type)
	assert.Equal(t, "_infinimail", txtRecord.Name)
	assert.Equal(t, "infinimail-verify=abc123xyz", txtRecord.Content)
	assert.Equal(t, 3600, txtRecord.TTL)
	assert.False(t, txtRecord.Proxied)
}

func TestExportRoute53(t *testing.T) {
	exporter := NewDNSExporter()
	guide := createTestDNSGuide()

	changeBatch, err := exporter.ExportRoute53(guide, "example.com")

	assert.NoError(t, err)
	assert.NotNil(t, changeBatch)
	assert.Contains(t, changeBatch.Comment, "example.com")
	assert.Len(t, changeBatch.Changes, 3)

	// Verify MX record
	mxChange := changeBatch.Changes[0]
	assert.Equal(t, "UPSERT", mxChange.Action)
	assert.Equal(t, "example.com.", mxChange.ResourceRecordSet.Name)
	assert.Equal(t, "MX", mxChange.ResourceRecordSet.Type)
	assert.Equal(t, 3600, mxChange.ResourceRecordSet.TTL)
	assert.Contains(t, mxChange.ResourceRecordSet.ResourceRecords[0].Value, "10 mail.infinimail.webrana.id.")

	// Verify A record
	aChange := changeBatch.Changes[1]
	assert.Equal(t, "UPSERT", aChange.Action)
	assert.Equal(t, "mail.example.com.", aChange.ResourceRecordSet.Name)
	assert.Equal(t, "A", aChange.ResourceRecordSet.Type)
	assert.Equal(t, "103.123.45.67", aChange.ResourceRecordSet.ResourceRecords[0].Value)

	// Verify TXT record
	txtChange := changeBatch.Changes[2]
	assert.Equal(t, "UPSERT", txtChange.Action)
	assert.Equal(t, "_infinimail.example.com.", txtChange.ResourceRecordSet.Name)
	assert.Equal(t, "TXT", txtChange.ResourceRecordSet.Type)
	assert.Contains(t, txtChange.ResourceRecordSet.ResourceRecords[0].Value, "infinimail-verify=abc123xyz")
}

func TestExportCSV(t *testing.T) {
	exporter := NewDNSExporter()
	guide := createTestDNSGuide()

	result := exporter.ExportCSV(guide, "example.com")

	lines := strings.Split(strings.TrimSpace(result), "\n")
	assert.Len(t, lines, 4) // Header + 3 records

	// Verify header
	assert.Equal(t, "Type,Name,Value,Priority,TTL", lines[0])

	// Verify MX record
	assert.Contains(t, lines[1], "MX")
	assert.Contains(t, lines[1], "example.com")
	assert.Contains(t, lines[1], "mail.infinimail.webrana.id")
	assert.Contains(t, lines[1], "10")
	assert.Contains(t, lines[1], "3600")

	// Verify A record
	assert.Contains(t, lines[2], "A")
	assert.Contains(t, lines[2], "mail.example.com")
	assert.Contains(t, lines[2], "103.123.45.67")

	// Verify TXT record
	assert.Contains(t, lines[3], "TXT")
	assert.Contains(t, lines[3], "_infinimail.example.com")
	assert.Contains(t, lines[3], "infinimail-verify=abc123xyz")
}

func TestExport_BIND(t *testing.T) {
	exporter := NewDNSExporter()
	guide := createTestDNSGuide()

	result, err := exporter.Export(guide, "example.com", ExportFormatBIND)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ExportFormatBIND, result.Format)
	assert.Equal(t, "example.com.zone", result.Filename)
	// Cloudflare-compatible format uses @ for root domain
	assert.Contains(t, result.Content.(string), "@\t3600\tIN\tMX")
}

func TestExport_Cloudflare(t *testing.T) {
	exporter := NewDNSExporter()
	guide := createTestDNSGuide()

	result, err := exporter.Export(guide, "example.com", ExportFormatCloudflare)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ExportFormatCloudflare, result.Format)
	assert.Equal(t, "example.com-cloudflare.json", result.Filename)

	// Verify it's valid JSON
	var records []CloudflareDNSRecord
	err = json.Unmarshal([]byte(result.Content.(string)), &records)
	assert.NoError(t, err)
	assert.Len(t, records, 3)
}

func TestExport_Route53(t *testing.T) {
	exporter := NewDNSExporter()
	guide := createTestDNSGuide()

	result, err := exporter.Export(guide, "example.com", ExportFormatRoute53)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ExportFormatRoute53, result.Format)
	assert.Equal(t, "example.com-route53.json", result.Filename)

	// Verify it's valid JSON
	var changeBatch Route53ChangeBatch
	err = json.Unmarshal([]byte(result.Content.(string)), &changeBatch)
	assert.NoError(t, err)
	assert.Len(t, changeBatch.Changes, 3)
}

func TestExport_CSV(t *testing.T) {
	exporter := NewDNSExporter()
	guide := createTestDNSGuide()

	result, err := exporter.Export(guide, "example.com", ExportFormatCSV)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ExportFormatCSV, result.Format)
	assert.Equal(t, "example.com-dns.csv", result.Filename)
	assert.Contains(t, result.Content.(string), "Type,Name,Value,Priority,TTL")
}

func TestExport_InvalidFormat(t *testing.T) {
	exporter := NewDNSExporter()
	guide := createTestDNSGuide()

	result, err := exporter.Export(guide, "example.com", "invalid")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid export format")
}

func TestDNSExportFormat_IsValid(t *testing.T) {
	tests := []struct {
		format   DNSExportFormat
		expected bool
	}{
		{ExportFormatBIND, true},
		{ExportFormatCloudflare, true},
		{ExportFormatRoute53, true},
		{ExportFormatCSV, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.format.IsValid())
		})
	}
}
