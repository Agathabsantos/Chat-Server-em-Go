# chat-go

Um servidor de chat TCP simples escrito em Go, focado no gerenciamento de estado concorrente usando canais (channels).

O projeto consiste em três componentes principais:
1.  **Servidor:** O "hub" central que gerencia conexões, apelidos e a distribuição de mensagens.
2.  **Cliente:** Um cliente de terminal para usuários humanos interagirem com o chat.
3.  **Bot:** Um cliente-exemplo de "bot" (chamado `Reverso`) que se conecta e responde a mensagens privadas invertendo o texto recebido.

## 🚀 Funcionalidades

* **Mensagens Públicas:** Todos os clientes (exceto bots) recebem mensagens enviadas publicamente.
* **Mensagens Privadas:** Envie mensagens diretas para um usuário (ou bot) específico usando `\msg @apelido texto...`.
* **Troca de Apelido:** Mude seu apelido a qualquer momento com `\changenick novo_apelido`.
* **Distinção de Bot/Humano:** O servidor sabe quais conexões são de bots (que se anunciam com `NICK [BOT]nome`) e não envia mensagens públicas para eles.
* **Validação de Apelido:** Apelidos devem ter entre 3 e 16 caracteres (letras, números ou `_`).
* **Gerenciamento de Estado Centralizado:** O `hub()` usa `select` em canais para evitar *race conditions* no acesso aos mapas de clientes.

## 📁 Estrutura do Código

Para que este projeto funcione, você deve salvar os três arquivos `main` separadamente. Sugerimos os seguintes nomes:

1.  `servidor.go` (O primeiro arquivo, que contém o `hub()`)
2.  `bot.go` (O segundo arquivo, que contém o `inverter()`)
3.  `cliente.go` (O terceiro arquivo, que contém o `fmt.Print("Escolha seu apelido...")`)

## ⚡ Como Executar

Você precisará de **três** janelas de terminal abertas no diretório onde salvou os arquivos.

### Terminal 1: Iniciar o Servidor

Primeiro, inicie o servidor. Ele ficará escutando na porta `9000`.

```
go run servidor.go
```

A saída deve ser:
```
Servidor escutando em 127.0.0.1:9000
```

### Terminal 2: Conectar o Cliente (Humano)

Em outra janela, inicie o cliente. Ele pedirá seu apelido.

```
go run cliente.go
```

Siga as instruções no terminal para escolher seu apelido e começar a conversar.

### Terminal 3: Conectar o Bot (Opcional)

Em uma terceira janela, inicie o bot. Por padrão, ele se chamará `Reverso`.

```
go run bot.go
```

Se quiser que o bot tenha um nome diferente, passe-o como argumento:

```
go run bot.go MeuBotInversor
```

## 🤖 Interagindo com o Bot

O bot `Reverso` só responde a mensagens privadas. No seu terminal de **Cliente** (Terminal 2), envie uma mensagem privada para ele:

```
\msg @Reverso ola mundo
```

O bot receberá a mensagem e responderá automaticamente:

```
@Reverso disse em privado: odnum alo
```

## 📝 Comandos do Cliente

Uma vez conectado como cliente (humano), você pode usar os seguintes comandos:

| Comando | Descrição |
| :--- | :--- |
| `\msg texto...` | Envia uma mensagem pública para todos. |
| `\msg @apelido texto...` | Envia uma mensagem privada para `@apelido`. |
| `\changenick novo_apelido` | Tenta trocar seu apelido atual para `novo_apelido`. |
| `\exit` | Desconecta do servidor. |
| `(qualquer outro texto)` | Também conta como uma mensagem pública. |
