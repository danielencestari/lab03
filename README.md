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
git clone https://github.com/devfullcycle/labs-auction-goexpert.git
cd labs-auction-goexpert
```

### 2. Configuração de Variáveis de Ambiente

O arquivo `.env` está localizado em `cmd/auction/.env`:

```env
MONGODB_URL=mongodb://mongodb:27017
MONGODB_DB=auctions
AUCTION_INTERVAL=5m
MAX_CONCURRENT_AUCTIONS=50
```

**Variáveis Disponíveis:**
- `AUCTION_INTERVAL`: Duração total do leilão (ex: 5m, 1h, 30s)
- `MAX_CONCURRENT_AUCTIONS`: Máximo de leilões simultâneos (padrão: 50)
- `MONGODB_URL`: URL de conexão com MongoDB
- `MONGODB_DB`: Nome do banco de dados

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

### Tratamento de Restart

- Ao inicializar a aplicação, todos os leilões ativos são automaticamente fechados
- Isso garante consistência após reinicializações não planejadas

### Validação de Lances

- Lances só são aceitos em leilões com status `Active`
- Sistema verifica tanto o status quanto o tempo do leilão

## 🧪 Testes

### Executar Todos os Testes
```bash
go test ./...
```

### Executar Testes Específicos
```bash
# Testes de fechamento automático
go test ./internal/infra/database/auction/ -v

# Teste específico
go test ./internal/infra/database/auction/ -run TestCreateAuctionWithAutoClose -v
```

### Testes Implementados

- ✅ **TestCreateAuctionWithAutoClose**: Valida fechamento automático por tempo
- ✅ **TestMaxConcurrentAuctions**: Valida limite de leilões simultâneos
- ✅ **TestUpdateAuctionStatus**: Valida atualização de status
- ✅ **TestConcurrentAuctionCreation**: Valida criação concorrente
- ✅ **TestAuctionDurationParsing**: Valida parsing de duração

## 🏗️ Arquitetura

### Estrutura do Projeto
```
├── cmd/auction/          # Aplicação principal
├── configuration/        # Configurações (DB, Logger)
├── internal/
│   ├── entity/          # Entidades de domínio
│   ├── usecase/         # Casos de uso
│   ├── infra/
│   │   ├── api/         # Controllers e validações
│   │   └── database/    # Repositórios
│   └── internal_error/  # Tratamento de erros
├── docker-compose.yml   # Orquestração
└── Dockerfile          # Container da aplicação
```

### Padrões Utilizados

- **Clean Architecture**: Separação clara de responsabilidades
- **Repository Pattern**: Abstração de acesso a dados
- **Dependency Injection**: Inversão de dependências
- **Concurrent Programming**: Goroutines para operações assíncronas

## 🔐 Tratamento de Erros

- **Padrão consistente**: Uso de `internal_error.InternalError`
- **Logs estruturados**: Info e Error com contexto
- **Validações**: Entrada e regras de negócio
- **Thread safety**: Mutexes para operações concorrentes

## 🚨 Limitações e Considerações

- **Máximo 50 leilões simultâneos** (configurável)
- **Restart fecha leilões ativos** (comportamento esperado)
- **MongoDB obrigatório** para persistência
- **Dependência de variáveis de ambiente** para configuração

## 🤝 Contribuição

1. Fork do projeto
2. Criar branch para feature (`git checkout -b feature/nova-feature`)
3. Commit das mudanças (`git commit -am 'Adiciona nova feature'`)
4. Push para branch (`git push origin feature/nova-feature`)
5. Criar Pull Request

## 📄 Licença

Este projeto é parte do curso GoExpert da Full Cycle.

---

**Desenvolvido com ❤️ em Go para o curso GoExpert da Full Cycle** 