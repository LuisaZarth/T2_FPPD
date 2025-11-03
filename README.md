# T2_FPPD
Trabalho 2 da disciplina de Fundamentos de Processamento Paralelo e Distribuído

Principais Componentes:
  - Servidor de Jogo: Gerencia sessões, estado do jogo (jogadores, posições, vidas) e requisições/respostas dos clientes. Não processa lógica de jogo nem possui interface gráfica.
 - Cliente do Jogo: Implementa a interface, lógica de movimentação e funcionamento. Conecta-se ao servidor para obter e enviar atualizações de estado, utilizando uma goroutine para buscas periódicas.
  
Comunicação (RPC):
- Toda a comunicação é iniciada pelos clientes.
- Implementa tratamento de erro com reexecução automática.
- Garante execução única (exactly-once) para comandos que modificam o estado do servidor, utilizando sequenceNumber e controle de comandos processados por cliente.


# Para rodar o jogo:
1. Rodar o servidor na pasta *servidor* usando:
        go run server.go
- servidor será iniciado em localhost:1234
2. rodar o cliente na pasta cliente usando:
        go run . localhost:1234 NomeDoJogador
3. iniciar outro jogador na pasta cliente (em um terminal diferente)
        gor un . localhost:1234 NomeDoJogador2