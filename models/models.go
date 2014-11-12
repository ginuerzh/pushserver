package models

import (
	"github.com/ginuerzh/sports/errors"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
)

var (
	databaseName = "sports"
	accountColl  = "accounts"
	MongoAddr    = "localhost:27017"
	mgoSession   *mgo.Session
)

type Location struct {
	Lat float64 `bson:"latitude" json:"latitude"`
	Lng float64 `bson:"longitude" json:"longitude"`
}

type Account struct {
	Id  string    `bson:"_id,omitempty" json:"-"`
	Loc *Location `bson:",omitempty" json:"-"`
}

func (this *Account) findOne(query interface{}) (bool, error) {
	var users []Account

	err := search(accountColl, query, nil, 0, 1, nil, nil, &users)
	if err != nil {
		return false, errors.NewError(errors.DbError, err.Error())
	}
	if len(users) > 0 {
		*this = users[0]
	}
	return len(users) > 0, nil
}

func (this *Account) FindByUserid(userid string) (bool, error) {
	return this.findOne(bson.M{"_id": userid})
}

func GetListByQuery(query bson.M) (total int, users []Account, err error) {
	if err := search(accountColl, query, nil, 0, 0, nil, nil, &users); err != nil {
		return 0, nil, errors.NewError(errors.DbError, err.Error())
	}
	return
}

func search(collection string, query interface{}, selector interface{},
	skip, limit int, sortFields []string, total *int, result interface{}) error {

	q := func(c *mgo.Collection) error {
		qy := c.Find(query)
		var err error

		if selector != nil {
			qy = qy.Select(selector)
		}

		if total != nil {
			if *total, err = qy.Count(); err != nil {
				return err
			}
		}

		if result == nil {
			return err
		}

		if limit > 0 {
			qy = qy.Limit(limit)
		}
		if skip > 0 {
			qy = qy.Skip(skip)
		}
		if len(sortFields) > 0 {
			qy = qy.Sort(sortFields...)
		}

		return qy.All(result)
	}

	if err := withCollection(collection, nil, q); err != nil {
		return errors.NewError(errors.DbError, err.Error())
	}
	return nil
}

func getSession() *mgo.Session {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial(MongoAddr)
		//log.Println(MongoAddr)
		if err != nil {
			log.Println(err) // no, not really
		}
	}
	return mgoSession.Clone()
}

func withCollection(collection string, safe *mgo.Safe, s func(*mgo.Collection) error) error {
	session := getSession()
	defer session.Close()

	session.SetSafe(safe)
	c := session.DB(databaseName).C(collection)
	return s(c)
}

func SaveToDB(collection string, o interface{}, safe bool) error {
	var err error
	insert := func(c *mgo.Collection) error {
		return c.Insert(o)
	}

	if safe {
		err = withCollection(collection, &mgo.Safe{}, insert)
	} else {
		err = withCollection(collection, nil, insert)
	}

	if err != nil {
		log.Println(err)
		return errors.NewError(errors.DbError, err.(*mgo.LastError).Error())
	}

	return nil
}
