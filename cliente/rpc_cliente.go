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
	maxRetries := 10
	retryDelay := 2 * time.Second

	log.Printf("[CLIENTE] Iniciando conexão ao servidor RPC %s para jogador %s", addr, playerID)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[CLIENTE] Tentativa %d/%d de conexão ao servidor %s", attempt, maxRetries, addr)

		cli, err := rpc.Dial("tcp", addr)
		if err == nil {
			log.Printf("[CLIENTE] ✅ Conexão TCP estabelecida com sucesso ao servidor %s", addr)

			rc := &RemoteClient{
				client:   cli,
				player:   playerID,
				PlayerID: playerID, // Garantir consistência
				remotos:  make(map[string]PlayerState),
			}

			// Tentar registrar o jogador
			log.Printf("[CLIENTE] Tentando registrar jogador %s...", playerID)
			if rc.registerWithReturn() {
				log.Printf("[CLIENTE] ✅ Cliente %s totalmente inicializado e registrado", playerID)
				go rc.pollingLoop()
				return rc
			} else {
				log.Printf("[CLIENTE] ❌ Falha no registro do jogador, fechando conexão")
				cli.Close()
				// Continua para próxima tentativa de conexão
			}
		} else {
			log.Printf("[CLIENTE] ❌ Falha na conexão TCP (tentativa %d/%d): %v", attempt, maxRetries, err)
		}

		// Se não é a última tentativa, espera antes de tentar novamente
		if attempt < maxRetries {
			log.Printf("[CLIENTE] ⏳ Aguardando %v antes da próxima tentativa...", retryDelay)
			time.Sleep(retryDelay)

			// Backoff exponencial limitado (máximo 10 segundos)
			retryDelay = time.Duration(float64(retryDelay) * 1.5)
			if retryDelay > 10*time.Second {
				retryDelay = 10 * time.Second
			}
		}
	}

	// Se chegou aqui, esgotou todas as tentativas
	log.Fatalf("[CLIENTE] 💀 ERRO CRÍTICO: Não foi possível conectar ao servidor %s após %d tentativas. Verifique se o servidor está rodando e se o endereço está correto.", addr, maxRetries)
	return nil // Nunca será executado devido ao log.Fatalf
}

// registerWithReturn tenta registrar o jogador e retorna true se bem-sucedido
func (rc *RemoteClient) registerWithReturn() bool {
	args := &RegisterArgs{PlayerID: rc.player}
	var rep RegisterReply
	maxRetries := 3

	for retries := 0; retries < maxRetries; retries++ {
		log.Printf("[CLIENTE] Tentativa %d/%d de registro do jogador %s", retries+1, maxRetries, rc.player)

		err := rc.client.Call("GameServer.RegisterPlayer", args, &rep)
		if err == nil && rep.OK {
			log.Printf("[CLIENTE] ✅ Jogador %s registrado com sucesso no servidor", rc.player)
			return true
		}

		if err != nil {
			log.Printf("[CLIENTE] ❌ Erro RPC no registro (tentativa %d/%d): %v", retries+1, maxRetries, err)
		} else {
			log.Printf("[CLIENTE] ❌ Servidor rejeitou registro (tentativa %d/%d): rep.OK=%v", retries+1, maxRetries, rep.OK)
		}

		if retries < maxRetries-1 {
			log.Printf("[CLIENTE] ⏳ Aguardando 1s antes da próxima tentativa de registro...")
			time.Sleep(time.Second)
		}
	}

	log.Printf("[CLIENTE] ❌ Falha ao registrar jogador %s após %d tentativas", rc.player, maxRetries)
	return false
}

// register mantém a interface original para compatibilidade (usa log.Fatalf)
func (rc *RemoteClient) register() {
	if !rc.registerWithReturn() {
		log.Fatalf("[CLIENTE] 💀 ERRO CRÍTICO: Falha ao registrar jogador %s", rc.player)
	}
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
		log.Printf("[CLIENTE] ❌ Erro ao atualizar estado (tentativa %d/%d): %v", retries+1, maxRetries, err)
		time.Sleep(time.Second)
	}

	log.Printf("[CLIENTE] ⚠️  AVISO: Falha ao atualizar posição após %d tentativas (seq=%d, pos=%d,%d)", maxRetries, seq, linha, col)
}

func (rc *RemoteClient) pollingLoop() {
	log.Printf("[CLIENTE] 🔄 Iniciando loop de polling para jogador %s", rc.player)

	consecutiveErrors := 0
	maxConsecutiveErrors := 5

	for {
		var gs GameState
		err := rc.client.Call("GameServer.GetGameState", new(struct{}), &gs)
		if err != nil {
			consecutiveErrors++
			log.Printf("[CLIENTE] ❌ Erro no polling (erro %d/%d): %v", consecutiveErrors, maxConsecutiveErrors, err)

			if consecutiveErrors >= maxConsecutiveErrors {
				log.Printf("[CLIENTE] 💀 ERRO CRÍTICO: Muitos erros consecutivos no polling (%d), encerrando cliente", consecutiveErrors)
				return
			}

			// Backoff progressivo em caso de erro
			errorDelay := time.Duration(consecutiveErrors) * time.Second
			log.Printf("[CLIENTE] ⏳ Aguardando %v antes da próxima tentativa de polling...", errorDelay)
			time.Sleep(errorDelay)
			continue
		}

		// Reset contador de erros em caso de sucesso
		if consecutiveErrors > 0 {
			log.Printf("[CLIENTE] ✅ Polling restaurado após %d erros", consecutiveErrors)
			consecutiveErrors = 0
		}

		rc.mu.Lock()
		previousCount := len(rc.remotos)
		rc.remotos = gs.Players
		currentCount := len(rc.remotos)
		rc.mu.Unlock()

		// Log mudanças no número de jogadores
		if currentCount != previousCount {
			log.Printf("[CLIENTE] 👥 Atualização de jogadores: %d -> %d jogadores online", previousCount, currentCount)
		}

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
	log.Printf("[CLIENTE] 🔌 Encerrando conexão do jogador %s...", rc.player)

	// Tentar avisar servidor antes de fechar (best effort)
	var empty struct{}
	err := rc.client.Call("GameServer.UnregisterPlayer", rc.PlayerID, &empty)
	if err != nil {
		log.Printf("[CLIENTE] ⚠️  Aviso: Erro ao desregistrar jogador no servidor: %v", err)
	} else {
		log.Printf("[CLIENTE] ✅ Jogador %s desregistrado do servidor", rc.player)
	}

	rc.client.Close()
	log.Printf("[CLIENTE] ✅ Conexão encerrada para jogador %s", rc.player)
}
