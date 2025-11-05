package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	const endereco = "127.0.0.1:9000"

	// Pergunta pelo apelido no terminal
	fmt.Print("Escolha seu apelido (3–16, letras/números/_): ")
	in := bufio.NewScanner(os.Stdin)
	if !in.Scan() {
		return
	}
	apelido := strings.TrimSpace(in.Text())
	if apelido == "" {
		fmt.Println("Apelido vazio. Encerrando.")
		return
	}

	conexao, err := net.Dial("tcp", endereco)
	if err != nil {
		log.Fatal(err)
	}
	defer conexao.Close()
	log.Printf("Conectado ao servidor %s\n", endereco)

	// Envia o nick no handshake
	fmt.Fprintf(conexao, "NICK %s\n", apelido)

	// Gorrotina: lê tudo que vem do servidor e imprime
	go func() {
		leitor := bufio.NewReader(conexao)
		for {
			linha, err := leitor.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Fprintln(os.Stderr, "Erro lendo do servidor:", err)
				}
				return
			}
			fmt.Print(linha)
			// reimprime prompt após mensagens do servidor
			fmt.Printf("%s> ", apelido)
		}
	}()

	fmt.Println(`Comandos:
  \msg texto...              -> mensagem pública
  \msg @apelido texto...     -> mensagem privada
  \changenick novo_apelido   -> trocar apelido
  \exit                      -> sair
  (qualquer texto sem comando também vira mensagem pública)`)
	fmt.Printf("%s> ", apelido)

	for in.Scan() {
		texto := in.Text()
		_, _ = fmt.Fprintln(conexao, texto)
		fmt.Printf("%s> ", apelido)
	}
}
