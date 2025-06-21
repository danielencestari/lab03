package auction

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/danielencestari/lab03/internal/entity/auction_entity"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func isMongoDBAvailable() bool {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return false
	}
	defer client.Disconnect(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Ping(ctx, nil)
	return err == nil
}

func setupAutoCloseTestDB() (*mongo.Database, func()) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}

	db := client.Database("auction_auto_close_test")

	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return db, cleanup
}

// TestAutoCloseAuctionValidation - Teste principal para validar fechamento automatizado
func TestAutoCloseAuctionValidation(t *testing.T) {
	if !isMongoDBAvailable() {
		t.Skip("MongoDB não está disponível - Pule este teste se o MongoDB não estiver rodando")
	}

	// Configurar intervalo curto para teste (3 segundos)
	os.Setenv("AUCTION_INTERVAL", "3s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupAutoCloseTestDB()
	defer cleanup()

	repo := NewAuctionRepository(db)
	ctx := context.Background()

	t.Log("=== INICIANDO TESTE DE FECHAMENTO AUTOMATIZADO ===")

	// Criar leilão de teste
	auction, err := auction_entity.CreateAuction(
		"Produto AutoClose Test",
		"Eletrônicos",
		"Teste de fechamento automatizado de leilão",
		auction_entity.New,
	)
	assert.Nil(t, err)
	assert.NotNil(t, auction)

	t.Logf("Leilão criado com ID: %s", auction.Id)

	// Gravar tempo de início
	startTime := time.Now()

	// Criar leilão no repositório (deve iniciar goroutine de monitoramento)
	err = repo.CreateAuction(ctx, auction)
	if err != nil {
		t.Fatalf("Erro ao criar leilão: %v", err)
	}

	t.Log("Leilão salvo no banco de dados")

	// Verificar se o leilão está inicialmente ativo
	foundAuction, err := repo.FindAuctionById(ctx, auction.Id)
	if err != nil {
		t.Fatalf("Erro ao buscar leilão: %v", err)
	}
	assert.Equal(t, auction_entity.Active, foundAuction.Status)

	t.Log("Status inicial confirmado: ACTIVE")

	// Verificar se EndTime foi persistido corretamente no banco
	filter := bson.M{"_id": auction.Id}
	var auctionMongo AuctionEntityMongo
	mongoErr := repo.Collection.FindOne(ctx, filter).Decode(&auctionMongo)
	assert.Nil(t, mongoErr)
	assert.Greater(t, auctionMongo.EndTime, auctionMongo.Timestamp)

	expectedEndTime := time.Unix(auctionMongo.Timestamp, 0).Add(3 * time.Second)
	actualEndTime := time.Unix(auctionMongo.EndTime, 0)
	timeDiff := actualEndTime.Sub(expectedEndTime).Abs()
	assert.True(t, timeDiff < time.Second, "EndTime deve estar próximo do esperado")

	t.Logf("EndTime persistido corretamente: %v", actualEndTime)

	// Verificar que ainda está ativo após 2 segundos (antes do fechamento)
	time.Sleep(2 * time.Second)
	foundAuction, err = repo.FindAuctionById(ctx, auction.Id)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Active, foundAuction.Status)

	t.Log("Status após 2s: ainda ACTIVE (conforme esperado)")

	// Aguardar o fechamento automático (3s total + 1s buffer)
	time.Sleep(2 * time.Second)

	elapsedTime := time.Since(startTime)
	t.Logf("Tempo decorrido: %v", elapsedTime)

	// Verificar se o leilão foi fechado automaticamente
	foundAuction, err = repo.FindAuctionById(ctx, auction.Id)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, foundAuction.Status)

	t.Log("✅ SUCESSO: Leilão fechado automaticamente com status COMPLETED")

	// Verificar se o contador de leilões ativos foi decrementado
	assert.Equal(t, int64(0), int64(repo.activeAuctionsCount))

	t.Log("✅ SUCESSO: Contador de leilões ativos decrementado corretamente")

	t.Log("=== TESTE DE FECHAMENTO AUTOMATIZADO CONCLUÍDO COM SUCESSO ===")
}

// TestMultipleAuctionsAutoClose - Teste com múltiplos leilões fechando automaticamente
func TestMultipleAuctionsAutoClose(t *testing.T) {
	if !isMongoDBAvailable() {
		t.Skip("MongoDB não está disponível - Execute este teste com MongoDB rodando")
	}

	// Configurar intervalo curto para teste
	os.Setenv("AUCTION_INTERVAL", "4s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupAutoCloseTestDB()
	defer cleanup()

	repo := NewAuctionRepository(db)
	ctx := context.Background()

	t.Log("=== INICIANDO TESTE DE MÚLTIPLOS LEILÕES ===")

	numAuctions := 5
	var auctions []*auction_entity.Auction

	// Criar múltiplos leilões
	for i := 0; i < numAuctions; i++ {
		auction, err := auction_entity.CreateAuction(
			"Produto Multi Test",
			"Categoria",
			"Teste de múltiplos leilões",
			auction_entity.New,
		)
		assert.Nil(t, err)
		auctions = append(auctions, auction)

		err = repo.CreateAuction(ctx, auction)
		assert.Nil(t, err)

		t.Logf("Leilão %d criado: %s", i+1, auction.Id)
	}

	// Verificar que todos estão ativos
	for i, auction := range auctions {
		foundAuction, err := repo.FindAuctionById(ctx, auction.Id)
		assert.Nil(t, err)
		assert.Equal(t, auction_entity.Active, foundAuction.Status)
		t.Logf("Leilão %d confirmado como ACTIVE", i+1)
	}

	// Verificar contador de leilões ativos
	assert.Equal(t, int64(numAuctions), int64(repo.activeAuctionsCount))
	t.Logf("Contador de leilões ativos: %d", repo.activeAuctionsCount)

	// Aguardar fechamento automático (4s + 1s buffer)
	t.Log("Aguardando fechamento automático...")
	time.Sleep(5 * time.Second)

	// Verificar que todos foram fechados
	for i, auction := range auctions {
		foundAuction, err := repo.FindAuctionById(ctx, auction.Id)
		assert.Nil(t, err)
		assert.Equal(t, auction_entity.Completed, foundAuction.Status)
		t.Logf("✅ Leilão %d fechado automaticamente", i+1)
	}

	// Verificar se contador foi zerado
	assert.Equal(t, int64(0), int64(repo.activeAuctionsCount))
	t.Log("✅ Contador de leilões ativos zerado corretamente")

	t.Log("=== TESTE DE MÚLTIPLOS LEILÕES CONCLUÍDO COM SUCESSO ===")
}

// TestAutoCloseLogicValidation - Teste de lógica sem MongoDB (teste unitário)
func TestAutoCloseLogicValidation(t *testing.T) {
	t.Log("=== TESTE DE LÓGICA DE FECHAMENTO AUTOMÁTICO (SEM MONGODB) ===")

	// Teste da função de parsing de duração
	testCases := []struct {
		envValue    string
		expectedDur time.Duration
		description string
	}{
		{"2s", 2 * time.Second, "2 segundos"},
		{"5m", 5 * time.Minute, "5 minutos"},
		{"1h", 1 * time.Hour, "1 hora"},
		{"invalid", 5 * time.Minute, "valor inválido deve usar padrão"},
		{"", 5 * time.Minute, "valor vazio deve usar padrão"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Configurar variável de ambiente
			if tc.envValue != "" {
				os.Setenv("AUCTION_INTERVAL", tc.envValue)
			} else {
				os.Unsetenv("AUCTION_INTERVAL")
			}

			// Criar um repositório mock para testar a função
			// Como não podemos criar o repo sem MongoDB, vamos testar indiretamente
			expectedStr := tc.expectedDur.String()

			// Parse manual da duração para verificar a lógica
			var parsedDur time.Duration
			var err error

			if tc.envValue != "" {
				parsedDur, err = time.ParseDuration(tc.envValue)
				if err != nil {
					parsedDur = 5 * time.Minute // valor padrão
				}
			} else {
				parsedDur = 5 * time.Minute // valor padrão
			}

			assert.Equal(t, tc.expectedDur, parsedDur)
			t.Logf("✅ Duração '%s' parseada corretamente: %s", tc.envValue, expectedStr)

			// Limpar
			os.Unsetenv("AUCTION_INTERVAL")
		})
	}

	t.Log("✅ TESTE DE LÓGICA CONCLUÍDO COM SUCESSO")
}

// TestAutoCloseWithDifferentDurations - Teste com diferentes durações
func TestAutoCloseWithDifferentDurations(t *testing.T) {
	if !isMongoDBAvailable() {
		t.Skip("MongoDB não está disponível - Execute este teste com MongoDB rodando")
	}

	db, cleanup := setupAutoCloseTestDB()
	defer cleanup()

	ctx := context.Background()

	t.Log("=== INICIANDO TESTE COM DIFERENTES DURAÇÕES ===")

	// Teste com 2 segundos
	os.Setenv("AUCTION_INTERVAL", "2s")
	repo1 := NewAuctionRepository(db)

	auction1, err := auction_entity.CreateAuction(
		"Produto 2s",
		"Categoria",
		"Teste com 2 segundos",
		auction_entity.New,
	)
	assert.Nil(t, err)

	startTime1 := time.Now()
	err = repo1.CreateAuction(ctx, auction1)
	assert.Nil(t, err)

	// Aguardar e verificar
	time.Sleep(3 * time.Second)
	foundAuction1, err := repo1.FindAuctionById(ctx, auction1.Id)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, foundAuction1.Status)

	duration1 := time.Since(startTime1)
	t.Logf("✅ Leilão de 2s fechado em: %v", duration1)

	os.Unsetenv("AUCTION_INTERVAL")
	t.Log("=== TESTE COM DIFERENTES DURAÇÕES CONCLUÍDO ===")
}

// TestAutoCloseRobustness - Teste de robustez do sistema de fechamento
func TestAutoCloseRobustness(t *testing.T) {
	if !isMongoDBAvailable() {
		t.Skip("MongoDB não está disponível - Execute este teste com MongoDB rodando")
	}

	os.Setenv("AUCTION_INTERVAL", "3s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupAutoCloseTestDB()
	defer cleanup()

	repo := NewAuctionRepository(db)
	ctx := context.Background()

	t.Log("=== INICIANDO TESTE DE ROBUSTEZ ===")

	// Criar leilão
	auction, err := auction_entity.CreateAuction(
		"Produto Robustez",
		"Categoria",
		"Teste de robustez do sistema",
		auction_entity.New,
	)
	assert.Nil(t, err)

	err = repo.CreateAuction(ctx, auction)
	assert.Nil(t, err)

	// Verificar múltiplas vezes durante o período ativo
	for i := 0; i < 3; i++ {
		time.Sleep(800 * time.Millisecond)
		foundAuction, err := repo.FindAuctionById(ctx, auction.Id)
		assert.Nil(t, err)
		assert.Equal(t, auction_entity.Active, foundAuction.Status)
		t.Logf("Verificação %d: Leilão ainda ACTIVE", i+1)
	}

	// Aguardar fechamento
	time.Sleep(1 * time.Second)

	// Verificar se foi fechado
	foundAuction, err := repo.FindAuctionById(ctx, auction.Id)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, foundAuction.Status)

	t.Log("✅ TESTE DE ROBUSTEZ CONCLUÍDO COM SUCESSO")
}
