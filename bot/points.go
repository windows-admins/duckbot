package main

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/storage"
	"log"
	"sort"
	"strings"
)

func updateScore(item string, operation string, guild string, isUser bool) int {
	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	tableClient := client.GetTableService()
	pointTable := tableClient.GetTableReference(storagePointTable)
	entity := pointTable.GetEntityReference(guild, strings.ToUpper(item))
	entityOptions := storage.GetEntityOptions{Select: []string{"Points", "isUser"}}
	entity.Get(30, storage.NoMetadata, &entityOptions)
	if entity.Properties["Points"] == nil {
		entity.Properties = map[string]interface{}{
			"Points": 0,
			"isUser": isUser,
		}
	}
	if operation == "--" || operation == "—" {
		log.Output(0, "Existing Record, Adding one point")
		entity.Properties["isUser"] = isUser
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
	if operation == "++" {
		log.Output(0, "Existing Record, removing one point")
		entity.Properties["isUser"] = isUser
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

func getTopInGuild(guild string, getMembers bool) []PointItem {
	client, err := storage.NewBasicClient(storageAccount, storageAccessToken)
	if err != nil {
		fmt.Printf("%s: \n", err)
	}
	tableClient := client.GetTableService()
	table := tableClient.GetTableReference(storagePointTable)
	queryOptions := storage.QueryOptions{Top: 1000, Filter: fmt.Sprintf("PartitionKey eq '%[1]s' and isUser eq %[2]t", guild, getMembers)}
	entities, err := table.QueryEntities(30, storage.MinimalMetadata, &queryOptions)
	itemList := []PointItem{}
	for {
		if err != nil {
			fmt.Println(err)
		} else {
			for i := range entities.Entities {
				itemList = append(itemList, PointItem{Item: entities.Entities[i].RowKey, Points: entities.Entities[i].Properties["Points"].(float64), IsUser: getMembers})
			}
		}
		if entities.QueryNextLink.NextLink == nil {
			break
		}
		entities, err = entities.NextResults(nil)
	}
	sort.SliceStable(itemList, func(i, j int) bool {
		return itemList[i].Points > itemList[j].Points
	})
	return itemList
}
