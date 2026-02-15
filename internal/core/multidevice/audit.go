package multidevice

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// AuditFilters defines criteria for querying audit entries.
type AuditFilters struct {
	DeviceID string
	Action   string
	Result   string
	Since    time.Time
	Until    time.Time
	Limit    int
}

// Query returns audit entries matching the given filters. This provides
// structured filtering beyond what ListEntries offers.
func (a *AuditLogger) Query(filters AuditFilters) ([]*AuditEntry, error) {
	path, err := a.auditLogPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*AuditEntry{}, nil
		}
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer f.Close()

	var results []*AuditEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry AuditEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if !matchesAuditFilters(&entry, &filters) {
			continue
		}

		results = append(results, &entry)

		if filters.Limit > 0 && len(results) >= filters.Limit {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	return results, nil
}

// matchesAuditFilters checks if an audit entry matches the given filter criteria.
func matchesAuditFilters(entry *AuditEntry, filters *AuditFilters) bool {
	if filters.DeviceID != "" && entry.DeviceID != filters.DeviceID {
		return false
	}
	if filters.Action != "" && entry.Action != filters.Action {
		return false
	}
	if filters.Result != "" && entry.Result != filters.Result {
		return false
	}
	if !filters.Since.IsZero() && entry.Timestamp.Before(filters.Since) {
		return false
	}
	if !filters.Until.IsZero() && entry.Timestamp.After(filters.Until) {
		return false
	}
	return true
}
