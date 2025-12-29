package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"sort"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/api/response"
	"github.com/welldanyogia/webrana-infinimail-backend/internal/services"
)

// ACMELogHandler handles ACME log viewing endpoints
type ACMELogHandler struct {
	logger *services.ACMELogger
}

// NewACMELogHandler creates a new ACME log handler
func NewACMELogHandler() *ACMELogHandler {
	return &ACMELogHandler{
		logger: services.GetACMELogger(),
	}
}

// ListLogs returns a list of all ACME logs
// GET /api/acme/logs
func (h *ACMELogHandler) ListLogs(c echo.Context) error {
	logs := h.logger.GetAllLogs()

	// Sort by updated_at descending (most recent first)
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].UpdatedAt.After(logs[j].UpdatedAt)
	})

	return response.Success(c, logs)
}

// GetDomainLog returns the log for a specific domain
// GET /api/acme/logs/:domain
func (h *ACMELogHandler) GetDomainLog(c echo.Context) error {
	domain := c.Param("domain")
	if domain == "" {
		return response.BadRequest(c, "domain parameter is required")
	}

	log, exists := h.logger.GetDomainLog(domain)
	if !exists {
		return response.NotFound(c, "log not found for domain: "+domain)
	}

	return response.Success(c, log)
}


// ViewLogsHTML returns an HTML page for viewing ACME logs
// GET /acme/logs (browser-friendly)
func (h *ACMELogHandler) ViewLogsHTML(c echo.Context) error {
	logs := h.logger.GetAllLogs()

	// Sort by updated_at descending
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].UpdatedAt.After(logs[j].UpdatedAt)
	})

	html := generateLogsListHTML(logs)
	return c.HTML(http.StatusOK, html)
}

// ViewDomainLogHTML returns an HTML page for viewing a specific domain's ACME log
// GET /acme/logs/:domain (browser-friendly)
func (h *ACMELogHandler) ViewDomainLogHTML(c echo.Context) error {
	domain := c.Param("domain")
	if domain == "" {
		return c.HTML(http.StatusBadRequest, "<h1>Error</h1><p>Domain parameter is required</p>")
	}

	log, exists := h.logger.GetDomainLog(domain)
	if !exists {
		return c.HTML(http.StatusNotFound, "<h1>Not Found</h1><p>No log found for domain: "+template.HTMLEscapeString(domain)+"</p>")
	}

	html := generateDomainLogHTML(log)
	return c.HTML(http.StatusOK, html)
}

func generateLogsListHTML(logs []services.ACMEDomainLogSummary) string {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ACME Certificate Logs</title>
    <style>
        * { box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 1200px; 
            margin: 0 auto; 
            padding: 20px;
            background: #f5f5f5;
        }
        h1 { color: #333; border-bottom: 2px solid #007bff; padding-bottom: 10px; }
        .log-list { list-style: none; padding: 0; }
        .log-item { 
            background: white; 
            margin: 10px 0; 
            padding: 15px 20px; 
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .log-item:hover { box-shadow: 0 4px 8px rgba(0,0,0,0.15); }
        .domain { font-weight: bold; font-size: 1.1em; color: #333; }
        .status { 
            padding: 4px 12px; 
            border-radius: 20px; 
            font-size: 0.85em;
            font-weight: 500;
        }
        .status-success { background: #d4edda; color: #155724; }
        .status-failed { background: #f8d7da; color: #721c24; }
        .status-in_progress { background: #fff3cd; color: #856404; }
        .meta { color: #666; font-size: 0.9em; }
        a { color: #007bff; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .empty { text-align: center; padding: 40px; color: #666; }
        .refresh-btn {
            background: #007bff;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            cursor: pointer;
            margin-bottom: 20px;
        }
        .refresh-btn:hover { background: #0056b3; }
    </style>
</head>
<body>
    <h1>🔐 ACME Certificate Logs</h1>
    <button class="refresh-btn" onclick="location.reload()">🔄 Refresh</button>
    <p class="meta">Auto-refresh: <a href="javascript:setInterval(()=>location.reload(),10000)">Enable (10s)</a></p>
`

	if len(logs) == 0 {
		html += `<div class="empty">No ACME logs found. Certificate generation logs will appear here.</div>`
	} else {
		html += `<ul class="log-list">`
		for _, log := range logs {
			statusClass := "status-" + log.Status
			html += `<li class="log-item">
                <div>
                    <a href="/acme/logs/` + template.HTMLEscapeString(log.Domain) + `" class="domain">` + template.HTMLEscapeString(log.Domain) + `</a>
                    <div class="meta">
                        Started: ` + log.StartedAt.Format(time.RFC3339) + ` | 
                        Updated: ` + log.UpdatedAt.Format(time.RFC3339) + ` |
                        Entries: ` + intToString(log.EntryCount) + `
                    </div>
                </div>
                <span class="status ` + statusClass + `">` + template.HTMLEscapeString(log.Status) + `</span>
            </li>`
		}
		html += `</ul>`
	}

	html += `</body></html>`
	return html
}


func generateDomainLogHTML(log *services.ACMEDomainLog) string {
	statusClass := "status-" + log.Status

	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ACME Log: ` + template.HTMLEscapeString(log.Domain) + `</title>
    <style>
        * { box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            max-width: 1200px; 
            margin: 0 auto; 
            padding: 20px;
            background: #f5f5f5;
        }
        h1 { color: #333; }
        .back-link { margin-bottom: 20px; display: inline-block; }
        .header { 
            background: white; 
            padding: 20px; 
            border-radius: 8px;
            margin-bottom: 20px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .status { 
            padding: 4px 12px; 
            border-radius: 20px; 
            font-size: 0.85em;
            font-weight: 500;
            display: inline-block;
        }
        .status-success { background: #d4edda; color: #155724; }
        .status-failed { background: #f8d7da; color: #721c24; }
        .status-in_progress { background: #fff3cd; color: #856404; }
        .meta { color: #666; font-size: 0.9em; margin-top: 10px; }
        .entries { list-style: none; padding: 0; }
        .entry { 
            background: white; 
            margin: 8px 0; 
            padding: 12px 16px; 
            border-radius: 6px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            border-left: 4px solid #ccc;
        }
        .entry-INFO { border-left-color: #17a2b8; }
        .entry-WARNING { border-left-color: #ffc107; }
        .entry-ERROR { border-left-color: #dc3545; }
        .entry-DEBUG { border-left-color: #6c757d; }
        .entry-header { 
            display: flex; 
            justify-content: space-between; 
            align-items: center;
            margin-bottom: 8px;
        }
        .level { 
            font-weight: bold; 
            font-size: 0.8em;
            padding: 2px 8px;
            border-radius: 4px;
        }
        .level-INFO { background: #d1ecf1; color: #0c5460; }
        .level-WARNING { background: #fff3cd; color: #856404; }
        .level-ERROR { background: #f8d7da; color: #721c24; }
        .level-DEBUG { background: #e2e3e5; color: #383d41; }
        .timestamp { color: #666; font-size: 0.85em; }
        .step { color: #007bff; font-weight: 500; }
        .message { margin: 8px 0; }
        .details { 
            background: #f8f9fa; 
            padding: 10px; 
            border-radius: 4px;
            font-family: monospace;
            font-size: 0.9em;
            white-space: pre-wrap;
            word-break: break-all;
            margin-top: 8px;
        }
        a { color: #007bff; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .refresh-btn {
            background: #007bff;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 4px;
            cursor: pointer;
        }
        .refresh-btn:hover { background: #0056b3; }
        .filter-bar {
            margin: 15px 0;
            padding: 10px;
            background: white;
            border-radius: 6px;
        }
        .filter-bar label { margin-right: 15px; cursor: pointer; }
    </style>
</head>
<body>
    <a href="/acme/logs" class="back-link">← Back to all logs</a>
    <button class="refresh-btn" onclick="location.reload()" style="float:right">🔄 Refresh</button>
    
    <div class="header">
        <h1>🔐 ` + template.HTMLEscapeString(log.Domain) + `</h1>
        <span class="status ` + statusClass + `">` + template.HTMLEscapeString(log.Status) + `</span>
        <div class="meta">
            <strong>Started:</strong> ` + log.StartedAt.Format(time.RFC3339) + `<br>
            <strong>Last Updated:</strong> ` + log.UpdatedAt.Format(time.RFC3339) + `<br>
            <strong>Total Entries:</strong> ` + template.HTMLEscapeString(intToString(len(log.Entries))) + `
        </div>
    </div>

    <div class="filter-bar">
        <strong>Filter:</strong>
        <label><input type="checkbox" checked onchange="toggleLevel('INFO')"> INFO</label>
        <label><input type="checkbox" checked onchange="toggleLevel('WARNING')"> WARNING</label>
        <label><input type="checkbox" checked onchange="toggleLevel('ERROR')"> ERROR</label>
        <label><input type="checkbox" checked onchange="toggleLevel('DEBUG')"> DEBUG</label>
    </div>

    <ul class="entries">`

	for _, entry := range log.Entries {
		levelClass := "level-" + string(entry.Level)
		entryClass := "entry-" + string(entry.Level)

		html += `
        <li class="entry ` + entryClass + `" data-level="` + string(entry.Level) + `">
            <div class="entry-header">
                <span class="level ` + levelClass + `">` + string(entry.Level) + `</span>
                <span class="timestamp">` + entry.Timestamp.Format("15:04:05.000") + `</span>
            </div>
            <div class="step">` + template.HTMLEscapeString(entry.Step) + `</div>
            <div class="message">` + template.HTMLEscapeString(entry.Message) + `</div>`

		if entry.Details != nil {
			detailsJSON, _ := jsonMarshalIndent(entry.Details)
			html += `<div class="details">` + template.HTMLEscapeString(string(detailsJSON)) + `</div>`
		}

		html += `</li>`
	}

	html += `
    </ul>

    <script>
        function toggleLevel(level) {
            document.querySelectorAll('.entry[data-level="' + level + '"]').forEach(el => {
                el.style.display = el.style.display === 'none' ? 'block' : 'none';
            });
        }
    </script>
</body>
</html>`

	return html
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func jsonMarshalIndent(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
