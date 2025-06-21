# 🏆 Leilão GoExpert - Sistema de Leilões com Fechamento Automático

Sistema de leilões desenvolvido em Go com funcionalidade de fechamento automático utilizando goroutines e concorrência.

## ✨ Funcionalidades

- **Criação de leilões** com fechamento automático baseado em tempo
- **Sistema de lances (bids)** com validação de leilão ativo
- **Concorrência controlada** com limite de 50 leilões simultâneos
- **Monitoramento individual** por leilão usando goroutines
- **Fechamento automático** após restart da aplicação
- **API REST** completa para gerenciamento

## 🚀 Tecnologias Utilizadas

- **Go 1.20+** - Linguagem principal
- **Gin** - Framework web
- **MongoDB** - Banco de dados
- **Docker & Docker Compose** - Containerização
- **Goroutines** - Concorrência e paralelismo

## 📋 Pré-requisitos

- Docker e Docker Compose instalados
- Go 1.20+ (para desenvolvimento local)
- MongoDB (containerizado via Docker Compose)

## 🛠️ Configuração e Execução

### 1. Clonar o Repositório
```bash
git clone https://github.com/danielencestari/lab03.git
cd lab03
```

### 2. Configuração de Variáveis de Ambiente

Crie o arquivo `.env` no diretório `cmd/auction/`:

```bash
# Criar o arquivo de ambiente
touch cmd/auction/.env
```

**Exemplo do arquivo `cmd/auction/.env`:**

```env
# Configuração do MongoDB
MONGODB_URL=mongodb://mongodb:27017
MONGODB_DB=auctions

# Configuração dos Leilões
AUCTION_INTERVAL=5m
MAX_CONCURRENT_AUCTIONS=50
```

**Variáveis Disponíveis:**
- `AUCTION_INTERVAL`: Duração total do leilão (ex: 5m, 1h, 30s)
- `MAX_CONCURRENT_AUCTIONS`: Máximo de leilões simultâneos (padrão: 50)
- `MONGODB_URL`: URL de conexão com MongoDB
- `MONGODB_DB`: Nome do banco de dados

**Exemplos de `AUCTION_INTERVAL`:**
- `30s` - 30 segundos
- `5m` - 5 minutos  
- `1h` - 1 hora
- `2h30m` - 2 horas e 30 minutos

### 3. Executar com Docker Compose

```bash
# Iniciar todos os serviços
docker-compose up -d

# Verificar logs
docker-compose logs -f app

# Parar serviços
docker-compose down
```

### 4. Executar Localmente (Desenvolvimento)

```bash
# Instalar dependências
go mod tidy

# Executar aplicação
go run cmd/auction/main.go

# Executar testes
go test ./internal/infra/database/auction/...
```

## 📚 API Endpoints

### Leilões (Auctions)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `POST` | `/auction` | Criar novo leilão |
| `GET` | `/auction` | Listar leilões |
| `GET` | `/auction/:auctionId` | Buscar leilão por ID |
| `GET` | `/auction/winner/:auctionId` | Buscar lance vencedor |

### Lances (Bids)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `POST` | `/bid` | Criar novo lance |
| `GET` | `/bid/:auctionId` | Listar lances do leilão |

### Usuários (Users)

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| `GET` | `/user/:userId` | Buscar usuário por ID |

## 📝 Exemplo de Uso

### 1. Criar um Leilão
```bash
curl -X POST http://localhost:8080/auction \
  -H "Content-Type: application/json" \
  -d '{
    "product_name": "iPhone 15 Pro",
    "category": "Electronics",
    "description": "iPhone 15 Pro em excelente estado",
    "condition": 1
  }'
```

### 2. Criar um Lance
```bash
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '[{
    "user_id": "user123",
    "auction_id": "auction_id_aqui",
    "amount": 1500.00
  }]'
```

### 3. Buscar Leilões Ativos
```bash
curl "http://localhost:8080/auction?status=0"
```

## 🔧 Funcionalidade de Fechamento Automático

### Como Funciona

1. **Criação**: Quando um leilão é criado, uma goroutine individual é iniciada
2. **Timer**: A goroutine aguarda pelo tempo definido em `AUCTION_INTERVAL`
3. **Fechamento**: Após o tempo, o status é automaticamente alterado para `Completed`
4. **Controle**: Sistema mantém controle de leilões ativos (máximo 50)

### Tratamento de Restart (Estado Persistente)

- **Persistência de Estado**: Cada leilão tem seu tempo de término (`EndTime`) salvo no MongoDB
- **Recuperação Inteligente**: Ao reiniciar, o sistema recupera leilões ativos e recalcula o tempo restante
- **Continuidade**: Leilões continuam de onde pararam, mantendo o tempo correto
- **Leilões Expirados**: Leilões que expiraram durante a parada são fechados imediatamente

### Validação de Lances

- Lances só são aceitos em leilões com status `Active`
- Sistema verifica tanto o status quanto o tempo do leilão

## 🧪 Testes

### Pré-requisitos para Testes
Para executar os testes que validam o fechamento automatizado, é necessário ter o MongoDB rodando:

#### Opção 1: Usando Docker/Colima
```bash
# Se Docker não estiver rodando, iniciar Colima
colima start

# Iniciar MongoDB
docker run -d --name mongodb -p 27017:27017 mongo:latest

# Verificar se está rodando
docker ps | grep mongodb
```

#### Opção 2: Usando Docker Compose
```bash
# Iniciar apenas o MongoDB
docker-compose up -d mongodb
```

### Executar Todos os Testes
```bash
go test ./...
```

### Testes de Fechamento Automatizado

#### 🎯 Teste Principal - Validação de Fechamento Automático
```bash
# Teste completo de fechamento automatizado (3 segundos)
go test -v ./internal/infra/database/auction -run TestAutoCloseAuctionValidation
```

**O que este teste valida:**
- ✅ Leilão criado com status `Active`
- ✅ `EndTime` persistido corretamente no MongoDB
- ✅ Leilão permanece ativo durante o período configurado
- ✅ Leilão fechado automaticamente após 3 segundos
- ✅ Status alterado para `Completed`
- ✅ Contador de leilões ativos decrementado

#### 🔄 Teste de Múltiplos Leilões
```bash
# Teste com 5 leilões simultâneos (4 segundos cada)
go test -v ./internal/infra/database/auction -run TestMultipleAuctionsAutoClose
```

**O que este teste valida:**
- ✅ Criação de 5 leilões simultâneos
- ✅ Todos iniciados com status `Active`
- ✅ Contador de leilões ativos = 5
- ✅ Todos fechados automaticamente após 4 segundos
- ✅ Contador zerado após fechamento

#### ⚙️ Teste de Lógica (Sem MongoDB)
```bash
# Teste de parsing de duração - não requer MongoDB
go test -v ./internal/infra/database/auction -run TestAutoCloseLogicValidation
```

**O que este teste valida:**
- ✅ Parsing correto de durações: `2s`, `5m`, `1h`
- ✅ Tratamento de valores inválidos (usa padrão 5m)
- ✅ Tratamento de valores vazios (usa padrão 5m)

#### 🔧 Teste de Robustez
```bash
# Teste de robustez do sistema (3 segundos)
go test -v ./internal/infra/database/auction -run TestAutoCloseRobustness
```

**O que este teste valida:**
- ✅ Múltiplas verificações durante período ativo
- ✅ Status permanece `Active` durante o período
- ✅ Fechamento preciso após tempo configurado

#### 🕐 Teste de Diferentes Durações
```bash
# Teste com diferentes intervalos de tempo
go test -v ./internal/infra/database/auction -run TestAutoCloseWithDifferentDurations
```

### Executar Todos os Testes de Fechamento Automático
```bash
# Executar todos os testes de auto-close
go test -v ./internal/infra/database/auction -run TestAutoClose
```

### Executar Testes Específicos por Arquivo
```bash
# Testes do arquivo auto_close_test.go
go test -v ./internal/infra/database/auction/auto_close_test.go ./internal/infra/database/auction/create_auction.go

# Testes de recovery (requer MongoDB)
go test -v ./internal/infra/database/auction -run TestAuctionRecovery

# Todos os testes de auction
go test -v ./internal/infra/database/auction/
```

### 📊 Resultados Esperados dos Testes

Quando os testes são executados com sucesso, você verá saídas como:

```
=== RUN   TestAutoCloseAuctionValidation
    auto_close_test.go:62: === INICIANDO TESTE DE FECHAMENTO AUTOMATIZADO ===
    auto_close_test.go:74: Leilão criado com ID: cc56b7f6-52c6-4871-a609-93295a111a7c
    auto_close_test.go:85: Leilão salvo no banco de dados
    auto_close_test.go:94: Status inicial confirmado: ACTIVE
    auto_close_test.go:108: EndTime persistido corretamente: 2025-06-21 15:51:19 -0300 -03
    auto_close_test.go:116: Status após 2s: ainda ACTIVE (conforme esperado)
    auto_close_test.go:122: Tempo decorrido: 4.030200999s
    auto_close_test.go:129: ✅ SUCESSO: Leilão fechado automaticamente com status COMPLETED
    auto_close_test.go:134: ✅ SUCESSO: Contador de leilões ativos decrementado corretamente
    auto_close_test.go:136: === TESTE DE FECHAMENTO AUTOMATIZADO CONCLUÍDO COM SUCESSO ===
--- PASS: TestAutoCloseAuctionValidation (4.05s)
```

### 🔍 Troubleshooting dos Testes

#### Erro: "MongoDB não está disponível"
```bash
# Verificar se MongoDB está rodando
docker ps | grep mongodb

# Se não estiver, iniciar:
docker run -d --name mongodb -p 27017:27017 mongo:latest
```

#### Erro: "Cannot connect to Docker daemon"
```bash
# Iniciar Colima (macOS)
colima start

# Verificar Docker
docker --version
```

#### Testes que são pulados (SKIP)
Alguns testes são automaticamente pulados se o MongoDB não estiver disponível:
- `TestAuctionRecoveryAfterRestart`
- `TestExpiredAuctionRecovery`
- `TestAutoCloseAuctionValidation` (se MongoDB indisponível)

### 📝 Arquivos de Teste

| Arquivo | Descrição |
|---------|-----------|
| `auto_close_test.go` | **Testes principais de fechamento automático** |
| `create_auction_test.go` | Testes de criação e funcionalidades básicas |
| `auction_recovery_test.go` | Testes de recuperação após restart |

### 🚀 Exemplo Prático - Executando os Testes

#### Passo a Passo Completo:

```bash
# 1. Clonar o projeto (se ainda não fez)
git clone https://github.com/danielencestari/lab03.git
cd lab03/desafio_concorrrencia_leilao/lab03-leilao-goexpert

# 2. Iniciar infraestrutura
colima start                                    # Iniciar Colima (se no macOS)
docker run -d --name mongodb -p 27017:27017 mongo:latest  # Iniciar MongoDB

# 3. Verificar se MongoDB está rodando
docker ps | grep mongodb

# 4. Executar teste principal de fechamento automático
go test -v ./internal/infra/database/auction -run TestAutoCloseAuctionValidation

# 5. Executar todos os testes de fechamento automático
go test -v ./internal/infra/database/auction -run TestAutoClose

# 6. Executar todos os testes do módulo auction
go test -v ./internal/infra/database/auction/
```

#### Resultado Esperado:
```
✅ TestAutoCloseAuctionValidation - PASS (4.05s)
✅ TestMultipleAuctionsAutoClose - PASS (5.05s)  
✅ TestAutoCloseLogicValidation - PASS (0.00s)
✅ TestAutoCloseWithDifferentDurations - PASS (3.03s)
✅ TestAutoCloseRobustness - PASS (3.44s)
```

#### Limpeza (Opcional):
```bash
# Parar e remover container MongoDB
docker stop mongodb && docker rm mongodb

# Parar Colima
colima stop
```

## 🚨 Limitações e Considerações

- **Máximo 50 leilões simultâneos** (configurável)
- **Restart fecha leilões ativos** (comportamento esperado)
- **MongoDB obrigatório** para persistência
- **Dependência de variáveis de ambiente** para configuração
