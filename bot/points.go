package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

const storageOperationAttempts = 8

func updateScore(item string, operation string, guild string, isUser bool) (int, error) {
	if !isDiscordGuildID(guild) {
		return 0, errors.New("points require a Discord server")
	}

	delta, err := scoreDelta(operation)
	if err != nil {
		return 0, err
	}

	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	if err != nil {
		return 0, fmt.Errorf("create storage client: %w", err)
	}
	tableClient := client.GetTableService()
	table := tableClient.GetTableReference(storagePointTable)
	rowKey := strings.ToUpper(item)

	for attempt := 0; attempt < storageOperationAttempts; attempt++ {
		entity := table.GetEntityReference(guild, rowKey)
		getOptions := storage.GetEntityOptions{Select: []string{"Points", "isUser"}}
		getErr := entity.Get(30, storage.FullMetadata, &getOptions)
		if isStorageStatus(getErr, http.StatusTooManyRequests) {
			waitForStorageRetry(attempt)
			continue
		}

		if isStorageStatus(getErr, http.StatusNotFound) {
			nextPoints := float64(delta)
			entity.Properties = map[string]interface{}{
				"Points": nextPoints,
				"isUser": isUser,
			}
			if insertErr := entity.Insert(storage.NoMetadata, nil); insertErr != nil {
				if isStorageStatus(insertErr, http.StatusConflict) ||
					isStorageStatus(insertErr, http.StatusTooManyRequests) {
					waitForStorageRetry(attempt)
					continue
				}
				return 0, fmt.Errorf("insert score: %w", insertErr)
			}
			apiLeaderboardCache.invalidateGuild(guild)
			return int(nextPoints), nil
		}
		if getErr != nil {
			return 0, fmt.Errorf("get score: %w", getErr)
		}

		currentPoints, valueErr := pointPropertyValue(entity.Properties["Points"])
		if valueErr != nil {
			return 0, valueErr
		}
		nextPoints := currentPoints + float64(delta)
		entity.Properties = map[string]interface{}{
			"Points": nextPoints,
			"isUser": isUser,
		}
		if updateErr := entity.Update(false, nil); updateErr != nil {
			if isETagConflict(updateErr) ||
				isStorageStatus(updateErr, http.StatusConflict) ||
				isStorageStatus(updateErr, http.StatusTooManyRequests) {
				waitForStorageRetry(attempt)
				continue
			}
			return 0, fmt.Errorf("update score: %w", updateErr)
		}
		apiLeaderboardCache.invalidateGuild(guild)
		return int(nextPoints), nil
	}

	return 0, errors.New("score changed too frequently; try again")
}

func scoreDelta(operation string) (int, error) {
	switch operation {
	case "++":
		return 1, nil
	case "--", "—":
		return -1, nil
	default:
		return 0, fmt.Errorf("unsupported score operation %q", operation)
	}
}

func pointPropertyValue(value interface{}) (float64, error) {
	switch points := value.(type) {
	case int:
		return float64(points), nil
	case int32:
		return float64(points), nil
	case int64:
		return float64(points), nil
	case float32:
		return float64(points), nil
	case float64:
		return points, nil
	default:
		return 0, fmt.Errorf("score entity has invalid Points property type %T", value)
	}
}

func waitForStorageRetry(attempt int) {
	base := time.Duration(1<<attempt) * 10 * time.Millisecond
	jitter := time.Duration(rand.Int63n(int64(base) + 1))
	time.Sleep(base + jitter)
}

func getTopInGuild(guild string, getMembers bool) ([]PointItem, error) {
	if !isDiscordGuildID(guild) {
		return nil, errors.New("leaderboard requires a Discord server")
	}

	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	tableClient := client.GetTableService()
	table := tableClient.GetTableReference(storagePointTable)
	queryOptions := storage.QueryOptions{
		Top:    1000,
		Filter: fmt.Sprintf("PartitionKey eq '%s' and isUser eq %t", guild, getMembers),
	}
	entities, err := queryLeaderboardEntities(table, &queryOptions)
	if err != nil {
		return nil, fmt.Errorf("query leaderboard: %w", err)
	}

	itemList := make([]PointItem, 0, len(entities.Entities))
	for {
		for i := range entities.Entities {
			points, valueErr := pointPropertyValue(entities.Entities[i].Properties["Points"])
			if valueErr != nil {
				return nil, fmt.Errorf("read leaderboard item %s: %w", entities.Entities[i].RowKey, valueErr)
			}
			itemList = append(itemList, PointItem{
				Item:   entities.Entities[i].RowKey,
				Points: points,
				IsUser: getMembers,
			})
		}
		if entities.QueryNextLink.NextLink == nil {
			break
		}
		entities, err = nextLeaderboardEntities(entities)
		if err != nil {
			return nil, fmt.Errorf("query next leaderboard page: %w", err)
		}
	}

	sort.SliceStable(itemList, func(i, j int) bool {
		return itemList[i].Points > itemList[j].Points
	})
	return itemList, nil
}

func queryLeaderboardEntities(table *storage.Table, options *storage.QueryOptions) (*storage.EntityQueryResult, error) {
	var (
		entities *storage.EntityQueryResult
		err      error
	)
	for attempt := 0; attempt < storageOperationAttempts; attempt++ {
		entities, err = table.QueryEntities(30, storage.MinimalMetadata, options)
		if !isStorageStatus(err, http.StatusTooManyRequests) {
			return entities, err
		}
		waitForStorageRetry(attempt)
	}
	return nil, err
}

func nextLeaderboardEntities(entities *storage.EntityQueryResult) (*storage.EntityQueryResult, error) {
	var (
		next *storage.EntityQueryResult
		err  error
	)
	for attempt := 0; attempt < storageOperationAttempts; attempt++ {
		next, err = entities.NextResults(nil)
		if !isStorageStatus(err, http.StatusTooManyRequests) {
			return next, err
		}
		waitForStorageRetry(attempt)
	}
	return nil, err
}
