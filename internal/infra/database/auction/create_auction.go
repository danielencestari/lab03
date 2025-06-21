package auction

import (
	"context"
	"github.com/danielencestari/lab03/configuration/logger"
	"github.com/danielencestari/lab03/internal/entity/auction_entity"
	"github.com/danielencestari/lab03/internal/internal_error"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionRepository struct {
	Collection          *mongo.Collection
	activeAuctionsCount int64
	auctionCountMutex   *sync.Mutex
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	repo := &AuctionRepository{
		Collection:          database.Collection("auctions"),
		activeAuctionsCount: 0,
		auctionCountMutex:   &sync.Mutex{},
	}

	// Handle active auctions on restart
	go repo.handleActiveAuctionsOnRestart()

	return repo
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {

	// Check concurrent auctions limit
	if !ar.checkActiveAuctionsLimit() {
		logger.Error("Maximum concurrent auctions limit reached", nil)
		return internal_error.NewInternalServerError("Maximum concurrent auctions limit reached")
	}

	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}

	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	// Increment active auctions counter
	ar.auctionCountMutex.Lock()
	ar.activeAuctionsCount++
	ar.auctionCountMutex.Unlock()

	// Start individual auction monitor goroutine
	go ar.startIndividualAuctionMonitor(auctionEntity)

	logger.Info("Auction created successfully with auto-close monitoring")
	return nil
}

func (ar *AuctionRepository) UpdateAuctionStatus(
	ctx context.Context,
	auctionId string,
	status auction_entity.AuctionStatus) *internal_error.InternalError {

	filter := bson.M{"_id": auctionId}
	update := bson.M{"$set": bson.M{"status": status}}

	_, err := ar.Collection.UpdateOne(ctx, filter, update)
	if err != nil {
		logger.Error("Error trying to update auction status", err)
		return internal_error.NewInternalServerError("Error trying to update auction status")
	}

	return nil
}

func (ar *AuctionRepository) startIndividualAuctionMonitor(auctionEntity *auction_entity.Auction) {
	auctionDuration := ar.getAuctionDuration()
	timer := time.NewTimer(auctionDuration)

	<-timer.C

	// Create context for the update operation
	ctx := context.Background()

	// Update auction status to Completed
	if err := ar.UpdateAuctionStatus(ctx, auctionEntity.Id, auction_entity.Completed); err != nil {
		logger.Error("Error closing auction automatically", err)
		return
	}

	// Decrement active auctions counter
	ar.auctionCountMutex.Lock()
	ar.activeAuctionsCount--
	ar.auctionCountMutex.Unlock()

	logger.Info("Auction closed automatically due to timeout")
}

func (ar *AuctionRepository) checkActiveAuctionsLimit() bool {
	ar.auctionCountMutex.Lock()
	defer ar.auctionCountMutex.Unlock()

	maxAuctions := ar.getMaxConcurrentAuctions()
	return ar.activeAuctionsCount < maxAuctions
}

func (ar *AuctionRepository) handleActiveAuctionsOnRestart() {
	ctx := context.Background()

	// Find all active auctions
	filter := bson.M{"status": auction_entity.Active}
	cursor, err := ar.Collection.Find(ctx, filter)
	if err != nil {
		logger.Error("Error finding active auctions on restart", err)
		return
	}
	defer cursor.Close(ctx)

	var activeAuctions []AuctionEntityMongo
	if err := cursor.All(ctx, &activeAuctions); err != nil {
		logger.Error("Error decoding active auctions on restart", err)
		return
	}

	// Close all active auctions
	for _, auction := range activeAuctions {
		if err := ar.UpdateAuctionStatus(ctx, auction.Id, auction_entity.Completed); err != nil {
			logger.Error("Error closing auction on restart", err)
			continue
		}
	}

	if len(activeAuctions) > 0 {
		logger.Info("Active auctions closed after application restart")
	}
}

func (ar *AuctionRepository) getAuctionDuration() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		logger.Error("Error parsing AUCTION_INTERVAL, using default 5 minutes", err)
		return time.Minute * 5
	}
	return duration
}

func (ar *AuctionRepository) getMaxConcurrentAuctions() int64 {
	// Default to 50 if not set
	return 50
}
