package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func extractPlusMinusEventData(message string) []string {
	regex := *regexp.MustCompile(`@([!-=?-~]+)>?\s*(\+{2}|-{2}|—{1}|={2})`)
	res := regex.FindAllStringSubmatch(message, -1)

	println("Message: " + message)
	if res != nil {
		println("Extracted Subject: " + res[0][1] + " Extracted Operation: " + res[0][2])
		return []string{res[0][1], res[0][2]}
	}

	return nil
}

func dbCall() *mgo.Session {
	dialInfo := &mgo.DialInfo{
		Addrs:    []string{fmt.Sprintf("%s.documents.azure.com:10255", mongoDBDatabase)}, // Get HOST + PORT
		Timeout:  60 * time.Second,
		Database: mongoDBDatabase, // It can be anything
		Username: mongoDBDatabase, // Username
		Password: mongoDBPassword, // PASSWORD
		DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{})
		},
	}

	// Create a session which maintains a pool of socket connections
	session, err := mgo.DialWithInfo(dialInfo)

	if err != nil {
		fmt.Printf("Can't connect, go error %v\n", err)
		os.Exit(1)
	}

	// SetSafe changes the session safety mode.
	// If the safe parameter is nil, the session is put in unsafe mode, and writes become fire-and-forget,
	// without error checking. The unsafe mode is faster since operations won't hold on waiting for a confirmation.
	// http://godoc.org/labix.org/v2/mgo#Session.SetMode.
	session.SetSafe(&mgo.Safe{})

	return session.Clone()
}
func returnItemID(item string, guild string) []bson.ObjectId {

	itemS := PointItem{}
	collection := session.DB(mongoDBDatabase).C("PointItems")
	println(collection.FullName)
	err := collection.Find(bson.M{"item": item}).Select(bson.M{"guild": guild}).One(&itemS)
	if err != nil {
		log.Output(0, "Error locating record")
		return nil
	}

	return []bson.ObjectId{itemS.Id}
}
