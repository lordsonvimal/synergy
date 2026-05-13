package db

import "context"

// --- Model types ---

type Session struct {
	ID          string
	SessionType string // "play" | "class"
	Title       *string
	Status      string
	CreatedAt   int64
	StartedAt   *int64
	EndedAt     *int64
}

type Game struct {
	ID                 string
	SessionID          string
	WhiteParticipantID *string
	BlackParticipantID *string
	TimeControlNs      *int64
	IncrementNs        *int64
	Variant            string
	InitialFEN         *string
	Status             string
	Winner             *string
	CreatedAt          int64
	EndedAt            *int64
}

type Move struct {
	ID          int64
	GameID      string
	SessionID   string
	Seq         uint64
	UCI         string
	SAN         string
	FEN         string
	MoveNumber  int
	Color       string // "white" | "black"
	WRemNs      int64
	BRemNs      int64
	LagCompNs   int64
	ThinkTimeNs int64
	PlayedAt    int64
}

type GameEvent struct {
	ID         int64
	GameID     string
	SessionID  string
	EventType  string
	Payload    string
	OccurredAt int64
}

type Participant struct {
	ID          string
	SessionID   string
	Role        string
	DisplayName string
	Token       string
	JoinedAt    int64
	LeftAt      *int64
	Claimed     bool
}

type ChatMessage struct {
	ID            int64
	SessionID     string
	ParticipantID string
	Body          string
	CreatedAt     int64
}

// --- Store interfaces ---

type SessionStore interface {
	Create(ctx context.Context, s *Session) error
	Get(ctx context.Context, id string) (*Session, error)
	UpdateStatus(ctx context.Context, id, status string, endedAt *int64) error
}

type GameStore interface {
	Create(ctx context.Context, g *Game) error
	Get(ctx context.Context, id string) (*Game, error)
	ListBySession(ctx context.Context, sessionID string) ([]*Game, error)
	UpdateStatus(ctx context.Context, id, status string, winner *string, endedAt *int64) error
}

type MoveStore interface {
	InsertBatch(ctx context.Context, moves []*Move) error
	ListByGame(ctx context.Context, gameID string) ([]*Move, error)
}

type GameEventStore interface {
	InsertBatch(ctx context.Context, events []*GameEvent) error
	ListByGame(ctx context.Context, gameID string) ([]*GameEvent, error)
}

type ParticipantStore interface {
	Create(ctx context.Context, p *Participant) error
	GetByToken(ctx context.Context, token string) (*Participant, error)
	ListBySession(ctx context.Context, sessionID string) ([]*Participant, error)
	SetClaimed(ctx context.Context, id string, claimedAt int64) error
}

type ChatStore interface {
	Insert(ctx context.Context, m *ChatMessage) error
	ListBySession(ctx context.Context, sessionID string, limit int) ([]*ChatMessage, error)
}

type Repository interface {
	Sessions() SessionStore
	Games() GameStore
	Moves() MoveStore
	GameEvents() GameEventStore
	Participants() ParticipantStore
	Chat() ChatStore
	Close() error
}
