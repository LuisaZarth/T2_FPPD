// server.go — Servidor RPC do jogo (estado global + exactly-once)
package main

//pacotes padrão usados pelo servidor RPC
import (
	"errors"
	"log"
	"net"
	"net/rpc"
	"sync"

	"jogo/shared"
)

// Type aliases for shared RPC types
type PlayerState = shared.PlayerState
type GameState = shared.GameState
type RegisterArgs = shared.RegisterArgs
type RegisterReply = shared.RegisterReply
type MoveArgs = shared.MoveArgs
type MoveReply = shared.MoveReply

// estrutura do servidor com o estado global e meméria de "exactly-once"
type GameServer struct {
	mu        sync.Mutex             //proteger o acesso a 'players' e 'processed' em chamadas concorrentes
	players   map[string]PlayerState //mapa com a última posição conhecida de cada jogador
	processed map[string]int         // guarda o último SeqNum processado por cada jogador
}

// construtor: inicia mapas vazios
func NewGameServer() *GameServer {
	return &GameServer{
		players:   make(map[string]PlayerState),
		processed: make(map[string]int),
	}
}

// RPC: registrar jogador
// RegistrarPlayer: registra o playerID no servidor
func (s *GameServer) RegisterPlayer(args *RegisterArgs, rep *RegisterReply) error {
	if args == nil || args.PlayerID == "" { //valida PlayerID
		return errors.New("PlayerID vazio")
	}
	s.mu.Lock()                                 //usa lock para atualizar player
	if _, ok := s.players[args.PlayerID]; !ok { //não cria duplicado se já existir
		s.players[args.PlayerID] = PlayerState{ID: args.PlayerID}
	}
	s.mu.Unlock()
	rep.OK = true
	log.Printf("[RPC] RegisterPlayer: %s", args.PlayerID) //loga a operação no terminal para facilitar depuração
	return nil
}

// RPC: atualiza posição com exactly-once
// UpdatePlayerState: aplica nova posiçãoa do jogador com garantia exactly-once
func (s *GameServer) UpdatePlayerState(args *MoveArgs, rep *MoveReply) error {
	if args == nil || args.PlayerID == "" { //valida argumentos
		return errors.New("args inválidos")
	}
	s.mu.Lock() //lock para ler/atualizar 'player' e 'processed'
	defer s.mu.Unlock()

	if last, ok := s.processed[args.PlayerID]; ok && args.SeqNum <= last {
		rep.Applied = false // retransmissão: ignora (exactly-once)
		return nil
	}
	//aplica atualização e memoriza SeqNum
	s.players[args.PlayerID] = PlayerState{ID: args.PlayerID, Linha: args.Linha, Col: args.Col}
	s.processed[args.PlayerID] = args.SeqNum
	rep.Applied = true
	log.Printf("[RPC] UpdatePlayerState: %s -> (%d,%d) seq=%d", args.PlayerID, args.Linha, args.Col, args.SeqNum)
	return nil
}

func (s *GameServer) UnregisterPlayer(id string, _ *struct{}) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    delete(s.players, id)
    log.Printf("Jogador %s removido.\n", id)
    return nil
}



// RPC: obter snapshot do estado (polling dos clientes)
// GetGameState: devolve um snapshot consistente do estado atual
// copia o mapa 'players' sob lock para evitar data race
// cliente usa esse snapshot em uma goroutine de polling periódico
func (s *GameServer) GetGameState(_ *struct{}, rep *GameState) error {
	s.mu.Lock()
	cp := make(map[string]PlayerState, len(s.players))
	for k, v := range s.players {
		cp[k] = v
	}
	s.mu.Unlock()
	rep.Players = cp
	return nil
}

// ponto de entrada do servidor RPC
func main() {
	srv := NewGameServer()
	if err := rpc.Register(srv); err != nil {
		log.Fatal(err)
	}
	l, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("[RPC] Servidor escutando em :1234")
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Accept:", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}
