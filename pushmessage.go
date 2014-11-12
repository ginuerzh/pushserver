package main

import (
	"encoding/json"
	"flag"
	"github.com/garyburd/redigo/redis"
	"github.com/shevilangle/pushserver/errors"
	"github.com/shevilangle/pushserver/models"
	"github.com/shevilangle/transfer"
	"labix.org/v2/mgo/bson"
	"log"
	"time"
)

var (
	redisServer   string
	fromString    string
	toString      string
	eventCountStr string
	meterToLoc    float64
	eventColl     string
)

type MsgBody struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type EventData struct {
	Type string    `json:"type"`
	Id   string    `json:"pid"`
	From string    `json:"from"`
	To   string    `json:"to"`
	Body []MsgBody `json:"body"`
}

type Event struct {
	Id   bson.ObjectId `bson:"_id,omitempty" json:"-"`
	Type string        `json:"type"`
	Data EventData     `json:"push"`
	Time int64         `json:"time"`
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&fromString, "f", "sports:pubsub:notice", "listen channel")
	flag.StringVar(&toString, "t", "sports:pubsub:user:", "prefix of receiver channel")
	flag.StringVar(&redisServer, "r", "172.24.222.54:6379", "redis server")
	flag.StringVar(&models.MongoAddr, "m", "localhost:27017", "mongodb server")
	flag.Parse()
}

func main() {
	eventColl = "events"
	eventCountStr = "sports:user:info:"
	c := float64(10000) / float64(111319)
	meterToLoc = c
	log.Println("fromString is :", fromString)
	log.Println("toString is :", toString)
	p := &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisServer)
			if err != nil {
				log.Println(err)
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	t := transfer.NewTransfer(p, fromString, toString, eventCountStr, getNearbyUsers, saveDataToDB)
	t.Push()
	log.Println("over")
}

func getNearbyUsers(data []byte) (error, []string, []interface{}, string) {
	var event Event
	eventType := ""
	err := json.Unmarshal(data, &event)
	if err != nil {
		log.Println(err)
		return err, make([]string, 0), make([]interface{}, 0), eventType
	}

	userid := event.Data.From
	eventType = event.Data.Type
	log.Println("userid: ", userid)

	user := &models.Account{}
	if find, err := user.FindByUserid(userid); !find {
		if err == nil {
			err = errors.NewError(errors.NotExistsError, "user '"+userid+"' not exists")
		}
		log.Println("not find")
		return err, make([]string, 0), make([]interface{}, 0), eventType
	} else {
		query := bson.M{
			"loc": bson.M{
				"$near":        []float64{user.Loc.Lat, user.Loc.Lng},
				"$maxDistance": meterToLoc,
			},
			"_id": bson.M{
				"$ne": userid,
			},
		}
		_, u, e := models.GetListByQuery(query)
		if e != nil {
			log.Println("e :", e)
			return e, make([]string, 0), make([]interface{}, 0), eventType
		}
		usercount := len(u)
		log.Println("usercount is :", usercount)

		list := make([]string, usercount)
		for i, v := range u {
			list[i] = v.Id
		}

		events := make([]Event, usercount)
		es := make([]interface{}, usercount)
		for i, v := range u {
			events[i].Id = bson.NewObjectId()
			events[i].Type = event.Type
			events[i].Time = event.Time
			events[i].Data = event.Data
			//			log.Println("events[i].Data.from:", events[i].Data.From)
			events[i].Data.To = v.Id
			es[i] = events[i]
		}
		return nil, list, es, eventType
	}
}

func saveDataToDB(o interface{}) {
	e := models.SaveToDB(eventColl, o, true)
	if e != nil {
		log.Println("e:", e)
	}
}
