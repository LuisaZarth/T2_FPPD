// types.go - Shared RPC message types between client and server
package shared

// PlayerState represents the minimal player state maintained on the server
type PlayerState struct {
	ID    string
	Linha int
	Col   int
}

// GameState is the snapshot of game state returned to clients during polling
type GameState struct {
	Players map[string]PlayerState // id -> player state
}

// RegisterArgs is used to register a player on the server
type RegisterArgs struct{ PlayerID string }
type RegisterReply struct{ OK bool }

// MoveArgs is used to update player position
type MoveArgs struct {
	PlayerID string
	Linha    int
	Col      int
	SeqNum   int // exactly-once guarantee
}
type MoveReply struct{ Applied bool } // true: applied; false: duplicate (ignored)
