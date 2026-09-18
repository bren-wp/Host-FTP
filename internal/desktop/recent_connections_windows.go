//go:build windows

package desktop

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const maxRecentConnections = 5

type recentConnectionEntry struct {
	ProfileID   string
	Name        string
	Protocol    string
	Host        string
	Port        int
	Username    string
	ConnectedAt time.Time
}

func (a *app) rememberRecentConnection(host string) {
	if a == nil {
		return
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return
	}
	port, _ := strconv.Atoi(strings.TrimSpace(getText(a.port)))
	entry := recentConnectionEntry{
		Name:        host,
		Protocol:    a.protocolValue(),
		Host:        host,
		Port:        port,
		Username:    strings.TrimSpace(getText(a.user)),
		ConnectedAt: time.Now(),
	}
	if profile, ok := a.currentProfile(); ok && a.currentEndpointMatchesProfile(profile) {
		entry.ProfileID = profile.ID
		entry.Name = profile.Name
	}

	filtered := make([]recentConnectionEntry, 0, maxRecentConnections)
	for _, current := range a.recentConnections {
		if strings.EqualFold(current.Protocol, entry.Protocol) &&
			strings.EqualFold(current.Host, entry.Host) &&
			current.Port == entry.Port &&
			strings.EqualFold(current.Username, entry.Username) {
			continue
		}
		filtered = append(filtered, current)
		if len(filtered) >= maxRecentConnections-1 {
			break
		}
	}
	a.recentConnections = append([]recentConnectionEntry{entry}, filtered...)
}

func recentConnectionAge(now, connected time.Time) string {
	if connected.IsZero() {
		return "recently"
	}
	age := now.Sub(connected)
	if age < 0 {
		age = 0
	}
	switch {
	case age < time.Minute:
		return "just now"
	case age < time.Hour:
		minutes := int(age / time.Minute)
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case age < 24*time.Hour:
		hours := int(age / time.Hour)
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(age / (24 * time.Hour))
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func recentConnectionLabel(entry recentConnectionEntry, now time.Time) string {
	endpoint := strings.ToUpper(entry.Protocol) + " · " + entry.Host
	if entry.Port > 0 {
		endpoint += ":" + strconv.Itoa(entry.Port)
	}
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		name = entry.Host
	}
	return name + "   ·   " + endpoint + "   ·   " + recentConnectionAge(now, entry.ConnectedAt)
}
