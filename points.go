package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/storage"
)

func updateScore(item string, operation string, guild string) int {
	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	tableClient := client.GetTableService()
	pointTable := tableClient.GetTableReference(storagePointTable)
	entity := pointTable.GetEntityReference(guild, strings.ToUpper(item))
	entityOptions := storage.GetEntityOptions{Select: []string{"Points"}}
	entity.Get(30, storage.NoMetadata, &entityOptions)
	if entity.Properties["Points"] == nil {
		entity.Properties = map[string]interface{}{
			"Points": 0,
		}
	}
	if operation == "++" {
		log.Output(0, "Existing Record, Adding one point")
		switch entity.Properties["Points"].(type) {
		case int:
			entity.Properties["Points"] = entity.Properties["Points"].(int) + 1
		case float64:
			entity.Properties["Points"] = entity.Properties["Points"].(float64) + 1
		}
		err = entity.InsertOrMerge(nil)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Inserted! ")
		}
		switch entity.Properties["Points"].(type) {
		case int:
			return entity.Properties["Points"].(int)
		case float64:
			return int(entity.Properties["Points"].(float64))
		}
	}
	if operation == "--" {
		log.Output(0, "Existing Record, removing one point")
		switch entity.Properties["Points"].(type) {
		case int:
			entity.Properties["Points"] = entity.Properties["Points"].(int) - 1
		case float64:
			entity.Properties["Points"] = entity.Properties["Points"].(float64) - 1
		}
		err = entity.InsertOrMerge(nil)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Inserted! ")
		}
		switch entity.Properties["Points"].(type) {
		case int:
			return entity.Properties["Points"].(int)
		case float64:
			return int(entity.Properties["Points"].(float64))
		}
	}
	return 0
}
