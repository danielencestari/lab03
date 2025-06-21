package auction

import (
	"context"
	"github.com/danielencestari/lab03/internal/entity/auction_entity"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDB() (*mongo.Database, func()) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		panic(err)
	}

	db := client.Database("auction_test")

	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return db, cleanup
}

func TestCreateAuctionWithAutoClose(t *testing.T) {
	// Set short auction interval for testing
	os.Setenv("AUCTION_INTERVAL", "2s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupTestDB()
	defer cleanup()

	repo := NewAuctionRepository(db)
	ctx := context.Background()

	// Create test auction
	auction, err := auction_entity.CreateAuction(
		"Test Product",
		"Electronics",
		"Test description for auction",
		auction_entity.New,
	)
	assert.Nil(t, err)
	assert.NotNil(t, auction)

	// Create auction in repository
	err = repo.CreateAuction(ctx, auction)
	assert.Nil(t, err)

	// Verify auction is initially active
	foundAuction, err := repo.FindAuctionById(ctx, auction.Id)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Active, foundAuction.Status)

	// Wait for auto-close (2s + buffer)
	time.Sleep(3 * time.Second)

	// Verify auction is now completed
	foundAuction, err = repo.FindAuctionById(ctx, auction.Id)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, foundAuction.Status)
}

func TestMaxConcurrentAuctions(t *testing.T) {
	// Set very short auction interval for testing
	os.Setenv("AUCTION_INTERVAL", "10s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupTestDB()
	defer cleanup()

	repo := NewAuctionRepository(db)
	ctx := context.Background()

	// Create maximum number of auctions (50)
	var auctions []*auction_entity.Auction
	for i := 0; i < 50; i++ {
		auction, err := auction_entity.CreateAuction(
			"Test Product",
			"Electronics",
			"Test description for auction",
			auction_entity.New,
		)
		assert.Nil(t, err)
		auctions = append(auctions, auction)

		err = repo.CreateAuction(ctx, auction)
		assert.Nil(t, err)
	}

	// Try to create one more auction (should fail)
	extraAuction, err := auction_entity.CreateAuction(
		"Extra Product",
		"Electronics",
		"Test description for extra auction",
		auction_entity.New,
	)
	assert.Nil(t, err)

	err = repo.CreateAuction(ctx, extraAuction)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Maximum concurrent auctions limit reached")
}

func TestUpdateAuctionStatus(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()

	repo := NewAuctionRepository(db)
	ctx := context.Background()

	// Create test auction
	auction, err := auction_entity.CreateAuction(
		"Test Product",
		"Electronics",
		"Test description for auction",
		auction_entity.New,
	)
	assert.Nil(t, err)

	err = repo.CreateAuction(ctx, auction)
	assert.Nil(t, err)

	// Verify initial status
	foundAuction, err := repo.FindAuctionById(ctx, auction.Id)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Active, foundAuction.Status)

	// Update status to Completed
	err = repo.UpdateAuctionStatus(ctx, auction.Id, auction_entity.Completed)
	assert.Nil(t, err)

	// Verify status updated
	foundAuction, err = repo.FindAuctionById(ctx, auction.Id)
	assert.Nil(t, err)
	assert.Equal(t, auction_entity.Completed, foundAuction.Status)
}

func TestConcurrentAuctionCreation(t *testing.T) {
	os.Setenv("AUCTION_INTERVAL", "5s")
	defer os.Unsetenv("AUCTION_INTERVAL")

	db, cleanup := setupTestDB()
	defer cleanup()

	repo := NewAuctionRepository(db)
	ctx := context.Background()

	// Create multiple auctions concurrently
	numGoroutines := 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			auction, err := auction_entity.CreateAuction(
				"Concurrent Product",
				"Electronics",
				"Test description for concurrent auction",
				auction_entity.New,
			)
			if err != nil {
				results <- err
				return
			}

			err = repo.CreateAuction(ctx, auction)
			results <- err
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		if err == nil {
			successCount++
		}
	}

	// All should succeed as we're within the limit
	assert.Equal(t, numGoroutines, successCount)
}

func TestAuctionDurationParsing(t *testing.T) {
	db, cleanup := setupTestDB()
	defer cleanup()

	repo := NewAuctionRepository(db)

	// Test valid duration
	os.Setenv("AUCTION_INTERVAL", "10m")
	duration := repo.getAuctionDuration()
	assert.Equal(t, 10*time.Minute, duration)

	// Test invalid duration (should use default)
	os.Setenv("AUCTION_INTERVAL", "invalid")
	duration = repo.getAuctionDuration()
	assert.Equal(t, 5*time.Minute, duration)

	// Cleanup
	os.Unsetenv("AUCTION_INTERVAL")
}
