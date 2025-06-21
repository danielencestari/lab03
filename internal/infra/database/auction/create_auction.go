package auction

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/danielencestari/lab03/configuration/logger"
	"github.com/danielencestari/lab03/internal/entity/auction_entity"
	"github.com/danielencestari/lab03/internal/internal_error"

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
	EndTime     int64                           `bson:"end_time"`
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

	// Calcular tempo de término do leilão
	auctionDuration := ar.getAuctionDuration()
	endTime := auctionEntity.Timestamp.Add(auctionDuration)

	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
		EndTime:     endTime.Unix(),
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

func (ar *AuctionRepository) startIndividualAuctionMonitorWithEndTime(auctionId string, endTime time.Time) {
	now := time.Now()
	remainingTime := endTime.Sub(now)

	// Se o leilão já expirou, feche imediatamente
	if remainingTime <= 0 {
		ctx := context.Background()
		if err := ar.UpdateAuctionStatus(ctx, auctionId, auction_entity.Completed); err != nil {
			logger.Error("Error closing expired auction on restart", err)
		}
		logger.Info("Expired auction closed immediately on restart")
		return
	}

	timer := time.NewTimer(remainingTime)

	<-timer.C

	// Create context for the update operation
	ctx := context.Background()

	// Update auction status to Completed
	if err := ar.UpdateAuctionStatus(ctx, auctionId, auction_entity.Completed); err != nil {
		logger.Error("Error closing auction automatically", err)
		return
	}

	// Decrement active auctions counter
	ar.auctionCountMutex.Lock()
	ar.activeAuctionsCount--
	ar.auctionCountMutex.Unlock()

	logger.Info("Auction closed automatically after restart with remaining time")
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

	// Reiniciar leilões com base no tempo restante
	recoveredCount := 0
	for _, auction := range activeAuctions {
		endTime := time.Unix(auction.EndTime, 0)

		// Incrementar contador de leilões ativos
		ar.auctionCountMutex.Lock()
		if ar.activeAuctionsCount < ar.getMaxConcurrentAuctions() {
			ar.activeAuctionsCount++
			ar.auctionCountMutex.Unlock()

			// Iniciar goroutine com tempo restante
			go ar.startIndividualAuctionMonitorWithEndTime(auction.Id, endTime)
			recoveredCount++
		} else {
			ar.auctionCountMutex.Unlock()
			// Se exceder o limite, feche o leilão
			if err := ar.UpdateAuctionStatus(ctx, auction.Id, auction_entity.Completed); err != nil {
				logger.Error("Error closing auction due to limit on restart", err)
			}
		}
	}

	if len(activeAuctions) > 0 {
		logger.Info("Active auctions recovered after restart")
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
