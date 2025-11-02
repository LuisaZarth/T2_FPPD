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
	mu       sync.Mutex
	client   *rpc.Client
	seq      int
	player   string
	PlayerID string
	remotos  map[string]PlayerState
}

func NewRemoteClient(playerID, addr string) *RemoteClient {
	for {
		cli, err := rpc.Dial("tcp", addr)
		if err == nil {
			rc := &RemoteClient{
				client:   cli,
				player:   playerID,
				PlayerID: playerID, // Garantir consistência
				remotos:  make(map[string]PlayerState),
			}
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
	maxRetries := 3

	for retries := 0; retries < maxRetries; retries++ {
		err := rc.client.Call("GameServer.RegisterPlayer", args, &rep)
		if err == nil && rep.OK {
			log.Printf("Jogador %s registrado com sucesso", rc.player)
			return
		}
		log.Printf("Erro ao registrar jogador (tentativa %d/%d): %v", retries+1, maxRetries, err)
		time.Sleep(time.Second)
	}
	log.Fatalf("Falha ao registrar jogador após %d tentativas", maxRetries)
}

func (rc *RemoteClient) updateState(linha, col int) {
	// Incrementa sequência sob lock
	rc.mu.Lock()
	rc.seq++
	seq := rc.seq
	rc.mu.Unlock()

	args := &MoveArgs{
		PlayerID: rc.player,
		Linha:    linha,
		Col:      col,
		SeqNum:   seq,
	}
	var rep MoveReply
	maxRetries := 3

	for retries := 0; retries < maxRetries; retries++ {
		err := rc.client.Call("GameServer.UpdatePlayerState", args, &rep)
		
		if err == nil {
			if rep.Applied {
				// Comando aplicado com sucesso
				return
			}
			// Applied = false significa que foi duplicata (já processado)
			// Isso é esperado em retransmissões, não é erro
			log.Printf("Movimento seq=%d já processado (duplicata ignorada)", seq)
			return
		}

		// Houve erro na comunicação, tentar novamente com MESMO SeqNum
		log.Printf("Erro ao atualizar estado (tentativa %d/%d): %v", retries+1, maxRetries, err)
		time.Sleep(time.Second)
	}

	log.Printf("AVISO: Falha ao atualizar posição após %d tentativas (seq=%d)", maxRetries, seq)
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

func (rc *RemoteClient) close() {
	// Avisar servidor antes de fechar
	var empty struct{}
	rc.client.Call("GameServer.UnregisterPlayer", rc.PlayerID, &empty)
	rc.client.Close()
}