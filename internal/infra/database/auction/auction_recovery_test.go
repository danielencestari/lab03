package auction

import (
	"context"
	"testing"
	"time"

	"github.com/danielencestari/lab03/internal/entity/auction_entity"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDBForRecovery() (*mongo.Database, func()) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}

	db := client.Database("auction_recovery_test")

	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return db, cleanup
}

func TestAuctionRecoveryAfterRestart(t *testing.T) {
	t.Skip("Skipping MongoDB integration test - requires MongoDB server")

	db, cleanup := setupTestDBForRecovery()
	defer cleanup()

	ctx := context.Background()

	// Simular leilão que estava ativo antes do restart
	now := time.Now()
	endTime := now.Add(5 * time.Second) // 5 segundos no futuro

	// Inserir leilão ativo diretamente no banco (simulando estado antes do restart)
	auctionMongo := AuctionEntityMongo{
		Id:          "test-auction-recovery",
		ProductName: "Recovery Test Product",
		Category:    "Electronics",
		Description: "Test auction for recovery functionality",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   now.Unix(),
		EndTime:     endTime.Unix(),
	}

	collection := db.Collection("auctions")
	_, err := collection.InsertOne(ctx, auctionMongo)
	assert.Nil(t, err)

	// Criar repositório (isso vai triggerar a função de recovery)
	repo := NewAuctionRepository(db)

	// Dar tempo para o recovery processar
	time.Sleep(100 * time.Millisecond)

	// Verificar que o leilão ainda está ativo
	foundAuction, err := repo.FindAuctionById(ctx, "test-auction-recovery")
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Active, foundAuction.Status)

	// Aguardar o tempo restante + buffer
	time.Sleep(6 * time.Second)

	// Verificar que o leilão foi fechado automaticamente
	foundAuction, err = repo.FindAuctionById(ctx, "test-auction-recovery")
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, foundAuction.Status)
}

func TestExpiredAuctionRecovery(t *testing.T) {
	t.Skip("Skipping MongoDB integration test - requires MongoDB server")

	db, cleanup := setupTestDBForRecovery()
	defer cleanup()

	ctx := context.Background()

	// Simular leilão que já expirou antes do restart
	now := time.Now()
	endTime := now.Add(-1 * time.Hour) // 1 hora no passado (expirado)

	// Inserir leilão expirado diretamente no banco
	auctionMongo := AuctionEntityMongo{
		Id:          "test-auction-expired",
		ProductName: "Expired Test Product",
		Category:    "Electronics",
		Description: "Test auction that should be closed immediately",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active, // Ainda marcado como ativo
		Timestamp:   now.Add(-2 * time.Hour).Unix(),
		EndTime:     endTime.Unix(),
	}

	collection := db.Collection("auctions")
	_, err := collection.InsertOne(ctx, auctionMongo)
	assert.Nil(t, err)

	// Criar repositório (isso vai triggerar a função de recovery)
	repo := NewAuctionRepository(db)

	// Dar tempo para o recovery processar
	time.Sleep(200 * time.Millisecond)

	// Verificar que o leilão foi fechado imediatamente
	foundAuction, err := repo.FindAuctionById(ctx, "test-auction-expired")
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, foundAuction.Status)
}

func TestGetAuctionDuration(t *testing.T) {
	db, cleanup := setupTestDBForRecovery()
	defer cleanup()

	repo := NewAuctionRepository(db)

	// Test that default duration is returned when no env var is set
	duration := repo.getAuctionDuration()
	assert.Equal(t, 5*time.Minute, duration)
}
