package main

import (
	"log"
	"net/rpc"
	"sync"
	"time"

	"jogo/shared"
)

// Type aliases for shared RPC types
type PlayerState = shared.PlayerState
type GameState = shared.GameState
type RegisterArgs = shared.RegisterArgs
type RegisterReply = shared.RegisterReply
type MoveArgs = shared.MoveArgs
type MoveReply = shared.MoveReply

type RemoteClient struct {
	mu      sync.Mutex
	client  *rpc.Client
	seq     int
	player  string
	remotos map[string]PlayerState
}

func NewRemoteClient(playerID, addr string) *RemoteClient {
	for {
		cli, err := rpc.Dial("tcp", addr)
		if err == nil {
			rc := &RemoteClient{client: cli, player: playerID, remotos: make(map[string]PlayerState)}
			rc.register()
			go rc.pollingLoop()
			return rc
		}
		log.Println("Tentando reconectar ao servidor RPC...")
		time.Sleep(2 * time.Second)
	}
}

func (rc *RemoteClient) register() {
	args := &RegisterArgs{PlayerID: rc.player}
	var rep RegisterReply
	if err := rc.client.Call("GameServer.RegisterPlayer", args, &rep); err != nil {
		log.Println("Erro ao registrar jogador:", err)
	} else {
		log.Println("Jogador registrado:", rc.player)
	}
}

func (rc *RemoteClient) updateState(linha, col int) {
	rc.mu.Lock()
	rc.seq++
	seq := rc.seq
	rc.mu.Unlock()
	args := &MoveArgs{PlayerID: rc.player, Linha: linha, Col: col, SeqNum: seq}
	var rep MoveReply
	err := rc.client.Call("GameServer.UpdatePlayerState", args, &rep)
	if err != nil {
		log.Println("Erro RPC:", err)
	}
}

func (rc *RemoteClient) pollingLoop() {
	for {
		var gs GameState
		err := rc.client.Call("GameServer.GetGameState", new(struct{}), &gs)
		if err != nil {
			log.Println("Erro polling:", err)
			time.Sleep(time.Second)
			continue
		}
		rc.mu.Lock()
		rc.remotos = gs.Players
		rc.mu.Unlock()
		time.Sleep(200 * time.Millisecond)
	}
}

func (rc *RemoteClient) getRemotos() map[string]PlayerState {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	cp := make(map[string]PlayerState, len(rc.remotos))
	for k, v := range rc.remotos {
		cp[k] = v
	}
	return cp
}
