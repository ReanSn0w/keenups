package state

import (
	"maps"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Snapshot struct {
	Connected   bool              `json:"connected"`
	Ready       bool              `json:"ready"`
	DataOK      bool              `json:"data_ok"`
	ConnectedAt time.Time         `json:"connected_at,omitempty"`
	LastUpdate  time.Time         `json:"last_update,omitempty"`
	Variables   map[string]string `json:"variables"`
}

type Store struct {
	mu          sync.RWMutex
	warmup      time.Duration
	connected   bool
	dataOK      bool
	ready       bool
	connectedAt time.Time
	lastUpdate  time.Time
	variables   map[string]string
}

func New(warmup time.Duration) *Store {
	return &Store{warmup: warmup, variables: make(map[string]string)}
}

func (s *Store) Connected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = true
	s.dataOK = false
	s.ready = false
	s.connectedAt = time.Now()
	s.lastUpdate = time.Time{}
	s.variables = make(map[string]string)
}

func (s *Store) Disconnected() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = false
	s.dataOK = false
	s.ready = false
}

func (s *Store) Set(name, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.variables[name] = value
	s.lastUpdate = time.Now()
}

func (s *Store) SetDataOK(ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dataOK = ok
}

func (s *Store) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	variables := maps.Clone(s.variables)
	if !s.ready && s.connected && s.dataOK && plausible(variables) && time.Since(s.connectedAt) >= s.warmup {
		s.ready = true
	}
	ready := s.connected && s.dataOK && s.ready
	return Snapshot{
		Connected:   s.connected,
		Ready:       ready,
		DataOK:      s.dataOK,
		ConnectedAt: s.connectedAt,
		LastUpdate:  s.lastUpdate,
		Variables:   variables,
	}
}

func plausible(variables map[string]string) bool {
	statusTokens := strings.Fields(variables["ups.status"])
	onLine := contains(statusTokens, "OL")
	onBattery := contains(statusTokens, "OB")
	if !onLine && !onBattery {
		return false
	}
	checks := []struct {
		name     string
		min, max float64
	}{
		{"battery.charge", 0, 100},
		{"output.frequency", 45, 65},
		{"output.voltage", 150, 260},
		{"ups.load", 0, 100},
	}
	if onLine {
		checks = append(checks,
			struct {
				name     string
				min, max float64
			}{"input.frequency", 45, 65},
			struct {
				name     string
				min, max float64
			}{"input.voltage", 150, 280},
		)
	}
	for _, check := range checks {
		value, ok := variables[check.name]
		if !ok {
			continue
		}
		number, err := strconv.ParseFloat(value, 64)
		if err != nil || number < check.min || number > check.max {
			return false
		}
	}
	return true
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
