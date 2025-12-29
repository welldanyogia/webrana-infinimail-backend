//go:build api
// +build api

// Package api contains tests that run against a real backend server.
// Run with: go test -tags=api ./tests/api/... -v
// Requires backend to be running on localhost:8080
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	defaultBaseURL = "http://localhost:8081"
	defaultAPIKey  = "test-api-key-for-development-only-32chars"
)

// APITestSuite is the test suite for real API endpoint testing
type APITestSuite struct {
	suite.Suite
	baseURL string
	apiKey  string
	client  *http.Client

	// Test data IDs for cleanup
	createdDomainIDs  []uint
	createdMailboxIDs []uint
	createdMessageIDs []uint
}

func TestAPIEndpoints(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func (s *APITestSuite) SetupSuite() {
	s.baseURL = os.Getenv("API_BASE_URL")
	if s.baseURL == "" {
		s.baseURL = defaultBaseURL
	}

	s.apiKey = os.Getenv("API_KEY")
	if s.apiKey == "" {
		s.apiKey = defaultAPIKey
	}

	s.client = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Verify server is running
	resp, err := s.client.Get(s.baseURL + "/health")
	require.NoError(s.T(), err, "Backend server must be running on %s", s.baseURL)
	defer resp.Body.Close()
	require.Equal(s.T(), http.StatusOK, resp.StatusCode, "Health check should return 200")
}

func (s *APITestSuite) TearDownSuite() {
	// Cleanup created resources in reverse order
	for _, id := range s.createdMailboxIDs {
		s.deleteResource(fmt.Sprintf("/api/mailboxes/%d", id))
	}
	for _, id := range s.createdDomainIDs {
		s.deleteResource(fmt.Sprintf("/api/domains/%d", id))
	}
}

// Helper methods
func (s *APITestSuite) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, s.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	return s.client.Do(req)
}

func (s *APITestSuite) deleteResource(path string) {
	resp, _ := s.doRequest(http.MethodDelete, path, nil)
	if resp != nil {
		resp.Body.Close()
	}
}

func (s *APITestSuite) parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// =============================================================================
// HEALTH ENDPOINTS
// =============================================================================

func (s *APITestSuite) TestHealth_ReturnsHealthy() {
	resp, err := s.client.Get(s.baseURL + "/health")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "healthy", result["status"])
}

func (s *APITestSuite) TestReady_ReturnsReady() {
	resp, err := s.client.Get(s.baseURL + "/ready")
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "ready", result["status"])
}

// =============================================================================
// DOMAIN ENDPOINTS
// =============================================================================

func (s *APITestSuite) TestDomain_CRUD_Flow() {
	// CREATE
	createReq := map[string]interface{}{
		"name":      fmt.Sprintf("test-%d.com", time.Now().UnixNano()),
		"is_active": true,
	}

	resp, err := s.doRequest(http.MethodPost, "/api/domains", createReq)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var createResult struct {
		Success bool `json:"success"`
		Data    struct {
			ID       uint   `json:"id"`
			Name     string `json:"name"`
			IsActive bool   `json:"is_active"`
		} `json:"data"`
	}
	err = s.parseResponse(resp, &createResult)
	require.NoError(s.T(), err)
	assert.True(s.T(), createResult.Success)
	assert.NotZero(s.T(), createResult.Data.ID)
	assert.Equal(s.T(), createReq["name"], createResult.Data.Name)

	domainID := createResult.Data.ID
	s.createdDomainIDs = append(s.createdDomainIDs, domainID)

	// GET
	resp, err = s.doRequest(http.MethodGet, fmt.Sprintf("/api/domains/%d", domainID), nil)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var getResult struct {
		Success bool `json:"success"`
		Data    struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	err = s.parseResponse(resp, &getResult)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), domainID, getResult.Data.ID)

	// LIST
	resp, err = s.doRequest(http.MethodGet, "/api/domains", nil)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var listResult struct {
		Success bool          `json:"success"`
		Data    []interface{} `json:"data"`
	}
	err = s.parseResponse(resp, &listResult)
	require.NoError(s.T(), err)
	assert.True(s.T(), len(listResult.Data) > 0)

	// UPDATE
	updateReq := map[string]interface{}{
		"is_active": false,
	}
	resp, err = s.doRequest(http.MethodPut, fmt.Sprintf("/api/domains/%d", domainID), updateReq)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	// DELETE
	resp, err = s.doRequest(http.MethodDelete, fmt.Sprintf("/api/domains/%d", domainID), nil)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Remove from cleanup list since we deleted it
	s.createdDomainIDs = s.createdDomainIDs[:len(s.createdDomainIDs)-1]

	// Verify deleted
	resp, err = s.doRequest(http.MethodGet, fmt.Sprintf("/api/domains/%d", domainID), nil)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func (s *APITestSuite) TestDomain_Create_EmptyName_Returns400() {
	createReq := map[string]interface{}{
		"name": "",
	}

	resp, err := s.doRequest(http.MethodPost, "/api/domains", createReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusBadRequest, resp.StatusCode)
}

func (s *APITestSuite) TestDomain_Create_Duplicate_Returns409() {
	domainName := fmt.Sprintf("dup-test-%d.com", time.Now().UnixNano())
	createReq := map[string]interface{}{
		"name": domainName,
	}

	// First create
	resp, err := s.doRequest(http.MethodPost, "/api/domains", createReq)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var result struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	s.parseResponse(resp, &result)
	s.createdDomainIDs = append(s.createdDomainIDs, result.Data.ID)

	// Duplicate create
	resp, err = s.doRequest(http.MethodPost, "/api/domains", createReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusConflict, resp.StatusCode)
}

func (s *APITestSuite) TestDomain_Get_NotFound_Returns404() {
	resp, err := s.doRequest(http.MethodGet, "/api/domains/999999", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *APITestSuite) TestDomain_List_WithActiveOnlyFilter() {
	resp, err := s.doRequest(http.MethodGet, "/api/domains?active_only=true", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

// =============================================================================
// MAILBOX ENDPOINTS
// =============================================================================

func (s *APITestSuite) TestMailbox_CRUD_Flow() {
	// First create a domain
	domainName := fmt.Sprintf("mailbox-test-%d.com", time.Now().UnixNano())
	domainReq := map[string]interface{}{
		"name":      domainName,
		"is_active": true,
	}

	resp, err := s.doRequest(http.MethodPost, "/api/domains", domainReq)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var domainResult struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	s.parseResponse(resp, &domainResult)
	domainID := domainResult.Data.ID
	s.createdDomainIDs = append(s.createdDomainIDs, domainID)

	// CREATE MAILBOX
	mailboxReq := map[string]interface{}{
		"local_part": "testuser",
		"domain_id":  domainID,
	}

	resp, err = s.doRequest(http.MethodPost, "/api/mailboxes", mailboxReq)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var mailboxResult struct {
		Success bool `json:"success"`
		Data    struct {
			ID          uint   `json:"id"`
			LocalPart   string `json:"local_part"`
			FullAddress string `json:"full_address"`
		} `json:"data"`
	}
	err = s.parseResponse(resp, &mailboxResult)
	require.NoError(s.T(), err)
	assert.True(s.T(), mailboxResult.Success)
	assert.Equal(s.T(), "testuser", mailboxResult.Data.LocalPart)
	assert.Contains(s.T(), mailboxResult.Data.FullAddress, "@"+domainName)

	mailboxID := mailboxResult.Data.ID
	s.createdMailboxIDs = append(s.createdMailboxIDs, mailboxID)

	// GET MAILBOX
	resp, err = s.doRequest(http.MethodGet, fmt.Sprintf("/api/mailboxes/%d", mailboxID), nil)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// LIST MAILBOXES
	resp, err = s.doRequest(http.MethodGet, fmt.Sprintf("/api/mailboxes?domain_id=%d", domainID), nil)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var listResult struct {
		Success bool          `json:"success"`
		Data    []interface{} `json:"data"`
		Meta    struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	s.parseResponse(resp, &listResult)
	assert.True(s.T(), listResult.Meta.Total >= 1)

	// DELETE MAILBOX
	resp, err = s.doRequest(http.MethodDelete, fmt.Sprintf("/api/mailboxes/%d", mailboxID), nil)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	s.createdMailboxIDs = s.createdMailboxIDs[:len(s.createdMailboxIDs)-1]
}

func (s *APITestSuite) TestMailbox_CreateRandom() {
	// First create a domain
	domainName := fmt.Sprintf("random-test-%d.com", time.Now().UnixNano())
	domainReq := map[string]interface{}{
		"name":      domainName,
		"is_active": true,
	}

	resp, err := s.doRequest(http.MethodPost, "/api/domains", domainReq)
	require.NoError(s.T(), err)
	require.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var domainResult struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	s.parseResponse(resp, &domainResult)
	domainID := domainResult.Data.ID
	s.createdDomainIDs = append(s.createdDomainIDs, domainID)

	// CREATE RANDOM MAILBOX
	randomReq := map[string]interface{}{
		"domain_id": domainID,
	}

	resp, err = s.doRequest(http.MethodPost, "/api/mailboxes/random", randomReq)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), http.StatusCreated, resp.StatusCode)

	var result struct {
		Data struct {
			ID        uint   `json:"id"`
			LocalPart string `json:"local_part"`
		} `json:"data"`
	}
	s.parseResponse(resp, &result)
	assert.Len(s.T(), result.Data.LocalPart, 8) // Random local part is 8 chars
	s.createdMailboxIDs = append(s.createdMailboxIDs, result.Data.ID)
}

func (s *APITestSuite) TestMailbox_Create_InvalidDomain_Returns404() {
	mailboxReq := map[string]interface{}{
		"local_part": "test",
		"domain_id":  999999,
	}

	resp, err := s.doRequest(http.MethodPost, "/api/mailboxes", mailboxReq)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *APITestSuite) TestMailbox_List_WithPagination() {
	// Create domain first
	domainName := fmt.Sprintf("page-test-%d.com", time.Now().UnixNano())
	domainReq := map[string]interface{}{
		"name":      domainName,
		"is_active": true,
	}

	resp, err := s.doRequest(http.MethodPost, "/api/domains", domainReq)
	require.NoError(s.T(), err)

	var domainResult struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	s.parseResponse(resp, &domainResult)
	s.createdDomainIDs = append(s.createdDomainIDs, domainResult.Data.ID)

	// Test pagination params
	resp, err = s.doRequest(http.MethodGet, fmt.Sprintf("/api/mailboxes?domain_id=%d&limit=10&offset=0", domainResult.Data.ID), nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result struct {
		Meta struct {
			Limit  int `json:"limit"`
			Offset int `json:"offset"`
		} `json:"meta"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(s.T(), 10, result.Meta.Limit)
	assert.Equal(s.T(), 0, result.Meta.Offset)
}

// =============================================================================
// MESSAGE ENDPOINTS
// =============================================================================

func (s *APITestSuite) TestMessage_List_ForMailbox() {
	// Create domain and mailbox
	domainName := fmt.Sprintf("msg-test-%d.com", time.Now().UnixNano())
	domainReq := map[string]interface{}{
		"name":      domainName,
		"is_active": true,
	}

	resp, err := s.doRequest(http.MethodPost, "/api/domains", domainReq)
	require.NoError(s.T(), err)

	var domainResult struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	s.parseResponse(resp, &domainResult)
	s.createdDomainIDs = append(s.createdDomainIDs, domainResult.Data.ID)

	mailboxReq := map[string]interface{}{
		"local_part": "msgtest",
		"domain_id":  domainResult.Data.ID,
	}

	resp, err = s.doRequest(http.MethodPost, "/api/mailboxes", mailboxReq)
	require.NoError(s.T(), err)

	var mailboxResult struct {
		Data struct {
			ID uint `json:"id"`
		} `json:"data"`
	}
	s.parseResponse(resp, &mailboxResult)
	s.createdMailboxIDs = append(s.createdMailboxIDs, mailboxResult.Data.ID)

	// List messages (should be empty but endpoint should work)
	resp, err = s.doRequest(http.MethodGet, fmt.Sprintf("/api/mailboxes/%d/messages", mailboxResult.Data.ID), nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)

	var result struct {
		Success bool          `json:"success"`
		Data    []interface{} `json:"data"`
		Meta    struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.True(s.T(), result.Success)
}

func (s *APITestSuite) TestMessage_Get_NotFound_Returns404() {
	resp, err := s.doRequest(http.MethodGet, "/api/messages/999999", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *APITestSuite) TestMessage_MarkAsRead_NotFound_Returns404() {
	resp, err := s.doRequest(http.MethodPatch, "/api/messages/999999/read", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *APITestSuite) TestMessage_Delete_NotFound_Returns404() {
	resp, err := s.doRequest(http.MethodDelete, "/api/messages/999999/delete", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

// =============================================================================
// ATTACHMENT ENDPOINTS
// =============================================================================

func (s *APITestSuite) TestAttachment_Get_NotFound_Returns404() {
	resp, err := s.doRequest(http.MethodGet, "/api/attachments/999999", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *APITestSuite) TestAttachment_Download_NotFound_Returns404() {
	resp, err := s.doRequest(http.MethodGet, "/api/attachments/999999/download", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

func (s *APITestSuite) TestAttachment_List_MessageNotFound_Returns404() {
	resp, err := s.doRequest(http.MethodGet, "/api/messages/999999/attachments", nil)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusNotFound, resp.StatusCode)
}

// =============================================================================
// AUTHENTICATION TESTS
// =============================================================================

func (s *APITestSuite) TestAuth_MissingAPIKey_Returns401() {
	req, err := http.NewRequest(http.MethodGet, s.baseURL+"/api/domains", nil)
	require.NoError(s.T(), err)
	// No Authorization header

	resp, err := s.client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (s *APITestSuite) TestAuth_InvalidAPIKey_Returns401() {
	req, err := http.NewRequest(http.MethodGet, s.baseURL+"/api/domains", nil)
	require.NoError(s.T(), err)
	req.Header.Set("Authorization", "Bearer invalid-api-key")

	resp, err := s.client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusUnauthorized, resp.StatusCode)
}

func (s *APITestSuite) TestAuth_HealthEndpoint_NoAuthRequired() {
	// Health endpoint should work without auth
	req, err := http.NewRequest(http.MethodGet, s.baseURL+"/health", nil)
	require.NoError(s.T(), err)
	// No Authorization header

	resp, err := s.client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}

func (s *APITestSuite) TestAuth_ReadyEndpoint_NoAuthRequired() {
	// Ready endpoint should work without auth
	req, err := http.NewRequest(http.MethodGet, s.baseURL+"/ready", nil)
	require.NoError(s.T(), err)
	// No Authorization header

	resp, err := s.client.Do(req)
	require.NoError(s.T(), err)
	defer resp.Body.Close()

	assert.Equal(s.T(), http.StatusOK, resp.StatusCode)
}
