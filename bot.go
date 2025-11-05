package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
)

// regex para capturar: @Origem disse em privado: texto...
var rePriv = regexp.MustCompile(`^@([^ ]+)\s+disse em privado:\s+(.+)$`)

func inverter(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

func main() {
	const endereco = "127.0.0.1:9000"

	apelido := "Reverso"
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		apelido = strings.TrimSpace(os.Args[1])
	}

	conexao, err := net.Dial("tcp", endereco)
	if err != nil {
		log.Fatal(err)
	}
	defer conexao.Close()
	log.Printf("Bot conectado ao servidor %s como @%s\n", endereco, apelido)

	// Handshake: NICK com marcador [BOT]
	fmt.Fprintf(conexao, "NICK [BOT]%s\n", apelido)

	// Leitor do servidor
	leitor := bufio.NewReader(conexao)

	for {
		linha, err := leitor.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, "Erro lendo do servidor:", err)
			}
			return
		}
		linha = strings.TrimSpace(linha)
		if linha == "" {
			continue
		}

		// Ignorar mensagens de sistema
		if strings.HasPrefix(linha, "[sistema] ") {
			continue
		}

		// Tentar casar formato de privado: "@Origem disse em privado: texto"
		m := rePriv.FindStringSubmatch(linha)
		if len(m) == 3 {
			origem := m[1]
			texto := m[2]
			resp := inverter(texto)

			fmt.Printf("Mensagem de: @%s\nRecebi: %s\nResposta: %s\n", origem, texto, resp)

			// responder PRIVADO ao remetente
			fmt.Fprintf(conexao, "\\msg @%s %s\n", origem, resp)
		}
		// Qualquer outra linha que não seja privada é ignorada pelo bot
	}
}
