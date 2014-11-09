/*
	The mongo package is a very simple wrapper around the labix.org/v2/mgo
	package. It's purpose is to allow you to do CRUD operations with very
	little code. It's not exhaustive and not meant to do everything for you.
*/
package mongo

import (
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"

	"errors"
	"fmt"
	"reflect"
	"time"
)

var (
	mgoSession *mgo.Session
	servers    string
	database   string
	NoPtr      = errors.New("You must pass in a pointer")
)

// Set the mongo servers and the database
func SetServers(servers, db string) error {
	var err error

	database = db

	mgoSession, err = mgo.Dial(servers)
	return err
}

// Insert a single record. Must pass in a pointer to a struct. The struct must
// contain an Id field of type bson.ObjectId.
func Insert(records ...interface{}) error {
	for _, rec := range records {
		if !isPtr(rec) {
			return NoPtr
		}

		if err := addNewFields(rec); err != nil {
			return err
		}

		s, err := GetSession()
		if err != nil {
			return err
		}
		defer s.Close()

		coll := GetColl(s, typeName(rec))
		err = coll.Insert(rec)
		if err != nil {
			return err
		}
	}

	return nil
}

// Find one or more records. If a single struct is passed in we'll return one record.
// If a slice is passed in all records will be returned. Must pass in a pointer to a
// struct or slice of structs.
func Find(i interface{}, q bson.M) error {
	if !isPtr(i) {
		return NoPtr
	}

	s, err := GetSession()
	if err != nil {
		return err
	}
	defer s.Close()

	coll := GetColl(s, typeName(i))

	query := coll.Find(q)

	if isSlice(reflect.TypeOf(i)) {
		err = query.All(i)
	} else {
		err = query.One(i)
	}
	return err
}

// Find a single record by id. Must pass a pointer to a struct.
func FindById(i interface{}, id string) error {
	return Find(i, bson.M{"_id": id})
}

// Updates a record. Uses the Id to identify the record to update. Must pass in a pointer
// to a struct.
func Update(i interface{}) error {
	if !isPtr(i) {
		return NoPtr
	}

	err := addCurrentDateTime(i, "UpdatedAt")
	if err != nil {
		return err
	}

	s, err := GetSession()
	if err != nil {
		return err
	}
	defer s.Close()

	id, err := getObjIdFromStruct(i)
	if err != nil {
		return err
	}

	return GetColl(s, typeName(i)).Update(bson.M{"_id": id}, i)
}

// Deletes a record. Uses the Id to identify the record to delete. Must pass in a pointer
// to a struct.
func Delete(i interface{}) error {
	if !isPtr(i) {
		return NoPtr
	}

	s, err := GetSession()
	if err != nil {
		return err
	}
	defer s.Close()

	id, err := getObjIdFromStruct(i)
	if err != nil {
		return err
	}

	return GetColl(s, typeName(i)).RemoveId(id)
}

// Returns a Mongo session. You must call Session.Close() when you're done.
func GetSession() (*mgo.Session, error) {
	var err error

	if mgoSession == nil {
		mgoSession, err = mgo.Dial(servers)
		if err != nil {
			return nil, err
		}
	}

	return mgoSession.Clone(), nil
}

// We pass in the session because that is a clone of the original and the
// caller will need to close it when finished.
func GetColl(session *mgo.Session, coll string) *mgo.Collection {
	return session.DB(database).C(coll)
}

func getObjIdFromStruct(i interface{}) (bson.ObjectId, error) {
	v := reflect.ValueOf(i)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return bson.ObjectId(""), errors.New("Can't delete record. Type must be a struct.")
	}

	f := v.FieldByName("Id")
	if f.Kind() == reflect.Ptr {
		f = f.Elem()
	}

	return f.Interface().(bson.ObjectId), nil
}

func isPtr(i interface{}) bool {
	return reflect.ValueOf(i).Kind() == reflect.Ptr
}

func typeName(i interface{}) string {
	t := reflect.TypeOf(i)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if isSlice(t) {
		t = t.Elem()

		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
	}

	return t.Name()
}

// returns true if the interface is a slice
func isSlice(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Slice
}

func addNewFields(i interface{}) error {
	err := addId(i)
	if err != nil {
		return err
	}

	if err := addCurrentDateTime(i, "CreatedAt"); err != nil {
		return err
	}

	return addCurrentDateTime(i, "UpdatedAt")
}

func addCurrentDateTime(i interface{}, name string) error {
	if !hasStructField(i, name) {
		return nil
	}

	now := time.Now()

	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	f := v.FieldByName(name)
	if f.Kind() == reflect.Ptr {
		f = f.Elem()
	}

	if reflect.TypeOf(now) != f.Type() {
		return fmt.Errorf("%v must be time.Time type.", name)
	}

	if !f.CanSet() {
		return fmt.Errorf("Couldn't set time for field: %v", name)
	}

	f.Set(reflect.ValueOf(now))

	return nil
}

func hasStructField(i interface{}, field string) bool {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return false
	}

	_, found := t.FieldByName(field)
	return found
}

func addId(i interface{}) error {
	v := reflect.ValueOf(i)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return errors.New("Record must be a struct")
	}

	f := v.FieldByName("Id")
	if f.Kind() == reflect.Ptr {
		f = f.Elem()
	}

	if f.Kind() == reflect.String {
		if !f.Interface().(bson.ObjectId).Valid() {
			id := reflect.ValueOf(bson.NewObjectId())
			f.Set(id)
		}
	}

	return nil
}
