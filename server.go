package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
)

// -------- Tipos e canais do hub --------

type Cliente struct {
	conexao net.Conn
	saida   chan string // tudo que o servidor quer enviar para este cliente
	apelido string
	ehBot   bool
}

type PedidoEntrada struct {
	cliente  *Cliente
	resposta chan error
}

type PedidoTrocaNick struct {
	cliente  *Cliente
	novo     string
	resposta chan error
}

type PedidoPrivado struct {
	origem  *Cliente
	destino string
	texto   string
}

var (
	mensagensPublicas = make(chan string)          // já formatadas (ex.: "@ana disse: oi")
	pedidoEntrada     = make(chan PedidoEntrada)   // registrar novo cliente (com apelido)
	pedidoSaida       = make(chan *Cliente)        // remoção de cliente
	pedidoTrocaNick   = make(chan PedidoTrocaNick) // changenick
	pedidoPrivado     = make(chan PedidoPrivado)   // mensagem privada
)

// -------- Validação de apelido --------

var reApelido = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)

func validaApelido(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("apelido vazio")
	}
	if !reApelido.MatchString(s) {
		return errors.New("apelido inválido (use 3–16 chars: letras, números ou _ )")
	}
	return nil
}

// -------- Hub: estado central --------

func hub() {
	clientes := make(map[*Cliente]bool)     // conjunto de clientes
	porApelido := make(map[string]*Cliente) // unicidade de apelido

	broadcastSistema := func(msg string) {
		for c := range clientes {
			select {
			case c.saida <- "[sistema] " + msg:
			default:
				close(c.saida)
				delete(clientes, c)
				if porApelido[c.apelido] == c {
					delete(porApelido, c.apelido)
				}
			}
		}
	}

	for {
		select {
		case req := <-pedidoEntrada:
			cli := req.cliente
			apelido := strings.TrimSpace(cli.apelido)

			if err := validaApelido(apelido); err != nil {
				req.resposta <- err
				continue
			}
			if _, existe := porApelido[apelido]; existe {
				req.resposta <- fmt.Errorf("apelido já em uso: %s", apelido)
				continue
			}
			// registra
			clientes[cli] = true
			porApelido[apelido] = cli
			req.resposta <- nil
			if cli.ehBot {
				broadcastSistema(fmt.Sprintf("Bot @%s acabou de entrar", apelido))
				log.Printf("Bot %s chegou!\n", apelido)
			} else {
				broadcastSistema(fmt.Sprintf("Usuário @%s acabou de entrar", apelido))
				log.Printf("%s chegou!\n", apelido)
			}

		case cli := <-pedidoSaida:
			if _, ok := clientes[cli]; ok {
				delete(clientes, cli)
				if porApelido[cli.apelido] == cli {
					delete(porApelido, cli.apelido)
				}
				close(cli.saida)
				if cli.ehBot {
					broadcastSistema(fmt.Sprintf("Bot @%s saiu", cli.apelido))
					log.Printf("Bot %s se foi\n", cli.apelido)
				} else {
					broadcastSistema(fmt.Sprintf("Usuário @%s saiu", cli.apelido))
					log.Printf("%s se foi\n", cli.apelido)
				}
			}

		case req := <-pedidoTrocaNick:
			cli := req.cliente
			novo := strings.TrimSpace(req.novo)

			if err := validaApelido(novo); err != nil {
				req.resposta <- err
				continue
			}
			if _, existe := porApelido[novo]; existe {
				req.resposta <- fmt.Errorf("apelido já em uso: %s", novo)
				continue
			}
			antigo := cli.apelido
			// atualiza índices
			delete(porApelido, antigo)
			cli.apelido = novo
			porApelido[novo] = cli
			req.resposta <- nil
			if cli.ehBot {
				broadcastSistema(fmt.Sprintf("Bot @%s agora é @%s", antigo, novo))
				log.Printf("Bot %s agora é %s\n", antigo, novo)
			} else {
				broadcastSistema(fmt.Sprintf("Usuário @%s agora é @%s", antigo, novo))
				log.Printf("%s agora é %s\n", antigo, novo)
			}

		case msg := <-mensagensPublicas:
			log.Println(msg)
			// envia a todos os **humanos** (bots NÃO recebem público)
			for c := range clientes {
				if c.ehBot {
					continue
				}
				select {
				case c.saida <- msg:
				default:
					close(c.saida)
					delete(clientes, c)
					if porApelido[c.apelido] == c {
						delete(porApelido, c.apelido)
					}
				}
			}

		case pv := <-pedidoPrivado:
			dest := strings.TrimSpace(pv.destino)
			alvo, ok := porApelido[dest]
			if !ok {
				// avisa o remetente que o alvo não existe
				select {
				case pv.origem.saida <- fmt.Sprintf("[erro] Usuário @%s não encontrado", dest):
				default:
				}
				continue
			}
			// log no servidor (requisito do EP)
			log.Printf("@%s disse em privado para @%s: %s\n", pv.origem.apelido, dest, pv.texto)
			// envia só ao destinatário (humano ou bot)
			select {
			case alvo.saida <- fmt.Sprintf("@%s disse em privado: %s", pv.origem.apelido, pv.texto):
			default:
			}
		}
	}
}

// -------- Conexão individual --------

func tratarConexao(conexao net.Conn) {
	defer conexao.Close()
	endereco := conexao.RemoteAddr().String()
	log.Printf("Conexão aberta de %s\n", endereco)

	cliente := &Cliente{
		conexao: conexao,
		saida:   make(chan string, 16),
		apelido: "",
		ehBot:   false,
	}

	// escritor: tudo que cair em cliente.saida vai para a conexão
	go func() {
		for msg := range cliente.saida {
			fmt.Fprintln(cliente.conexao, msg)
		}
	}()

	// ---- Handshake de apelido (sem 2ª linha) ----
	leitorScanner := bufio.NewScanner(conexao)
	if !leitorScanner.Scan() {
		log.Printf("Conexão encerrada (sem nick) de %s\n", endereco)
		return
	}
	linha := strings.TrimSpace(leitorScanner.Text())
	if !strings.HasPrefix(linha, "NICK ") {
		fmt.Fprintln(conexao, `[erro] Primeiro envie: NICK <apelido>`)
		return
	}
	rawNick := strings.TrimSpace(strings.TrimPrefix(linha, "NICK "))

	// Detecta bot por prefixo no próprio apelido
	if strings.HasPrefix(strings.ToUpper(rawNick), "[BOT]") {
		cliente.ehBot = true
		cliente.apelido = strings.TrimSpace(rawNick[len("[BOT]"):])
	} else {
		cliente.apelido = rawNick
	}

	// registra no hub (dispara o broadcast "acabou de entrar")
	resp := make(chan error)
	pedidoEntrada <- PedidoEntrada{cliente: cliente, resposta: resp}
	if err := <-resp; err != nil {
		fmt.Fprintln(conexao, "[erro] Não foi possível entrar:", err)
		return
	}

	// mensagem de boas-vindas para o próprio cliente (depois do broadcast)
	if cliente.ehBot {
		fmt.Fprintf(conexao, "[sistema] Olá, bot @%s! Você entrou no chat.\n", cliente.apelido)
	} else {
		fmt.Fprintf(conexao, "[sistema] Olá, @%s! Você entrou no chat.\n", cliente.apelido)
	}

	// ---- Loop principal de leitura ----
	processarLinha := func(l string) bool {
		l = strings.TrimSpace(l)
		switch {
		case l == `\exit`:
			pedidoSaida <- cliente
			log.Printf("Conexão encerrada de %s (@%s)\n", endereco, cliente.apelido)
			return false

		case strings.HasPrefix(l, `\changenick`):
			novo := strings.TrimSpace(strings.TrimPrefix(l, `\changenick`))
			if err := validaApelido(novo); err != nil {
				fmt.Fprintln(conexao, "[erro] Não foi possível trocar:", err)
				return true
			}
			resp := make(chan error)
			pedidoTrocaNick <- PedidoTrocaNick{cliente: cliente, novo: novo, resposta: resp}
			if err := <-resp; err != nil {
				fmt.Fprintln(conexao, "[erro] Não foi possível trocar:", err)
			}
			return true

		case strings.HasPrefix(l, `\msg`):
			texto := strings.TrimSpace(strings.TrimPrefix(l, `\msg`))
			if texto == "" {
				fmt.Fprintln(conexao, "[erro] Mensagem vazia")
				return true
			}
			if strings.HasPrefix(texto, "@") {
				partes := strings.Fields(texto)
				if len(partes) < 2 {
					fmt.Fprintln(conexao, "[erro] Uso: \\msg @apelido texto...")
					return true
				}
				dest := strings.TrimPrefix(partes[0], "@")
				corpo := strings.TrimSpace(strings.TrimPrefix(texto, partes[0]))
				if corpo == "" {
					fmt.Fprintln(conexao, "[erro] Mensagem privada vazia")
					return true
				}
				pedidoPrivado <- PedidoPrivado{origem: cliente, destino: dest, texto: corpo}
			} else {
				mensagensPublicas <- fmt.Sprintf("@%s disse: %s", cliente.apelido, texto)
			}
			return true

		default:
			if l != "" {
				mensagensPublicas <- fmt.Sprintf("@%s disse: %s", cliente.apelido, l)
			}
			return true
		}
	}

	for leitorScanner.Scan() {
		if ok := processarLinha(leitorScanner.Text()); !ok {
			return
		}
	}
	if err := leitorScanner.Err(); err != nil {
		log.Printf("Erro lendo de %s (@%s): %v\n", endereco, cliente.apelido, err)
	}
	pedidoSaida <- cliente
	log.Printf("Conexão encerrada de %s (@%s)\n", endereco, cliente.apelido)
}

func main() {
	const endereco = "127.0.0.1:9000"

	// sobe o hub
	go hub()

	escuta, err := net.Listen("tcp", endereco)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Servidor escutando em %s\n", endereco)

	for {
		conn, err := escuta.Accept()
		if err != nil {
			log.Println("Erro no Accept:", err)
			continue
		}
		go tratarConexao(conn)
	}
}
