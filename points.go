package main

import (
	"log"

	"gopkg.in/mgo.v2/bson"
)

// PointItem represents a document in the collection

func updateScore(item string, operation string, guild string) []bson.ObjectId {

	collection := session.DB(mongoDBDatabase).C("PointItems")
	theID := returnItemID(item, guild)
	println(theID)
	if theID != nil {

		if operation == "++" {
			log.Output(0, "Existing Record, Adding one point")
			err := collection.Update(bson.M{"_id": bson.ObjectId(theID[0])}, bson.M{"$inc": bson.M{"points": 1}})
			if err != nil {
				log.Fatal("Error updating record: ", err)
				return nil
			}
			return []bson.ObjectId{theID[0]}
		}
		if operation == "--" {
			log.Output(0, "Existing Record, Removing one point")
			err := collection.Update(bson.M{"_id": theID[0]}, bson.M{"$inc": bson.M{"points": -1}})
			if err != nil {
				log.Fatal("Error updating record: ", err)
				return nil
			}
			return []bson.ObjectId{theID[0]}
		}

	} else {
		newID := bson.NewObjectId()
		log.Output(0, "New Record, Creating and Adding one point")
		if operation == "++" {

			err := collection.Insert(&PointItem{
				Id:     newID,
				Item:   item,
				Guild:  guild,
				Points: 1,
			})
			if err != nil {
				log.Fatal("Error updating record: ", err)
				return nil
			}
			return []bson.ObjectId{newID}
		}
		if operation == "--" {
			log.Output(0, "New Record, Creating and removing one point")
			err := collection.Insert(&PointItem{
				Id:     newID,
				Item:   item,
				Guild:  guild,
				Points: -1,
			})
			if err != nil {
				log.Fatal("Error updating record: ", err)
				return nil
			}
			return []bson.ObjectId{newID}
		}
	}
	return nil
}

func getPointsByID(id bson.ObjectId) int {

	collection := session.DB(mongoDBDatabase).C("PointItems")
	item := PointItem{}
	err := collection.FindId(bson.ObjectId(id)).One(&item)
	if err != nil {
		log.Output(0, "Error locating record")
		return 0
	}
	return item.Points
}
