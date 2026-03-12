package components

import "time"

const PresenceType = "presence"

const (
	StatusOnline  = "online"
	StatusIdle    = "idle"
	StatusWorking = "working"
	StatusAway    = "away"
)

type Presence struct {
	Status        string
	Zone          string
	LastHeartbeat time.Time
}

func (p *Presence) ComponentType() string { return PresenceType }
