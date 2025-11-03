// main.go - Loop principal do jogo
package main

import (
	"fmt"
	"os"
)

func main() {

	if len(os.Args) != 3 {
		fmt.Println("Uso:", os.Args[0], " <servidor> <nome_do_jogador>")
		return
	}
	interfaceIniciar() // iniciar a interface gráfica
	defer interfaceFinalizar()

	servidor := os.Args[1]
	nomeJogador := os.Args[2]

	cliente := NewRemoteClient(nomeJogador, servidor)
	defer cliente.close() // Usando o método close() que desregistra E fecha conexão

	jogo := jogoNovo()
	if err := jogoCarregarMapa("mapa.txt", &jogo); err != nil {
		panic(err) // encerrar o programa se o mapa não puder ser carregado
	}
	cliente.updateState(jogo.PosY, jogo.PosX) // sincroniza a posição inicial do player

	loopPrincipal(&jogo, cliente)
	fmt.Println("Jogo encerrado") // mensagem de encerramento do jogo
}

func loopPrincipal(jogo *Jogo, cliente *RemoteClient) {
	// Desenha o estado inicial
	interfaceDesenharJogo(jogo, cliente)

	for {
		evento := interfaceLerEventoTeclado()
		if continuar := personagemExecutarAcao(evento, jogo, cliente); !continuar {
			break
		}
		interfaceDesenharJogo(jogo, cliente)
	}
}