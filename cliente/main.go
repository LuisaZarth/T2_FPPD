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
	defer cliente.client.Close() // fechar a conexão com o servidor ao encerrar o programa

	jogo := jogoNovo()
	if err := jogoCarregarMapa("mapa.txt", &jogo); err != nil {
		panic(err) // encerrar o programa se o mapa não puder ser carregado
	}
	cliente.updateState(jogo.PosY, jogo.PosX) // Sync initial position

	loopPrincipal(&jogo, cliente)
	fmt.Println("Jogo encerrado") // mensagem de encerramento do jogo

}
func loopPrincipal(jogo *Jogo, cliente *RemoteClient) {
	// Draw initial state
	interfaceDesenharJogo(jogo, cliente)

	for {
		evento := interfaceLerEventoTeclado()
		if continuar := personagemExecutarAcao(evento, jogo, cliente); !continuar {
			break
		}
		interfaceDesenharJogo(jogo, cliente)
	}
}
