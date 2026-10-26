/*
The mongo package is a very simple wrapper around the github.com/globalsign/mgo/bson
package. It's purpose is to allow you to do CRUD operations with very
little code. It's not exhaustive and not meant to do everything for you.
*/
package mongo

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"
)

var (
	mu         sync.Mutex // guards mgoSession, servers and database
	mgoSession *mgo.Session
	servers    string
	database   string
	NoPtr      = errors.New("You must pass in a pointer")
)

// Set the mongo servers and the database
func SetServers(srv, db string) error {
	mu.Lock()
	defer mu.Unlock()

	servers = srv
	database = db

	if mgoSession != nil {
		mgoSession.Close()
		mgoSession = nil
	}

	s, err := mgo.Dial(servers)
	if err != nil {
		return err
	}
	mgoSession = s
	return nil
}

// Insert one or more structs. Must pass in a pointer to a struct. The struct must
// contain an Id field of type bson.ObjectId with a tag of `bson:"_id"`.
func Insert(records ...any) error {
	for _, rec := range records {
		if !isPtr(rec) {
			return NoPtr
		}
	}

	s, err := GetSession()
	if err != nil {
		return err
	}
	defer s.Close()

	for _, rec := range records {
		if err := addNewFields(rec); err != nil {
			return err
		}

		coll := GetColl(s, typeName(rec))
		if err := coll.Insert(rec); err != nil {
			return err
		}
	}

	return nil
}

// Find one or more records. If a single struct is passed in we'll return one record.
// If a slice is passed in all records will be returned. Must pass in a pointer to a
// struct or slice of structs. Use sortFields to sort the results. See
// http://www.mongodb.org/display/DOCS/Sorting+and+Natural+Order for more info.
func Find(i any, q bson.M, sortFields ...string) error {
	if !isPtr(i) {
		return NoPtr
	}

	s, err := GetSession()
	if err != nil {
		return err
	}
	defer s.Close()

	coll := GetColl(s, typeName(i))

	query := coll.Find(q).Sort(sortFields...)

	if isSlice(reflect.TypeOf(i)) {
		err = query.All(i)
	} else {
		err = query.One(i)
	}
	return err
}

// Find a single record by id. Must pass a pointer to a struct.
func FindById(i any, id string) error {
	if !bson.IsObjectIdHex(id) {
		return fmt.Errorf("invalid ObjectId hex: %q", id)
	}
	return Find(i, bson.M{"_id": bson.ObjectIdHex(id)})
}

// Updates a record. Uses the Id to identify the record to update. Must pass in a pointer
// to a struct.
func Update(i any) error {
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
func Delete(i any) error {
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

// Does a count on the collection for the struct that is passed in.
func Count(i any) (int, error) {
	s, err := GetSession()
	if err != nil {
		return 0, err
	}
	defer s.Close()

	coll := GetColl(s, typeName(i))

	return coll.Count()
}

// Returns a Mongo session. You must call Session.Close() when you're done.
func GetSession() (*mgo.Session, error) {
	mu.Lock()
	defer mu.Unlock()

	if mgoSession == nil {
		s, err := mgo.Dial(servers)
		if err != nil {
			return nil, err
		}
		mgoSession = s
	}

	return mgoSession.Clone(), nil
}

// We pass in the session because that is a clone of the original and the
// caller will need to close it when finished.
func GetColl(session *mgo.Session, coll string) *mgo.Collection {
	mu.Lock()
	db := database
	mu.Unlock()
	return session.DB(db).C(coll)
}

func getObjIdFromStruct(i any) (bson.ObjectId, error) {
	v := reflect.ValueOf(i)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return bson.ObjectId(""), errors.New("Record must be a struct")
	}

	f := v.FieldByName("Id")
	if f.Kind() == reflect.Ptr {
		f = f.Elem()
	}
	if !f.IsValid() {
		return bson.ObjectId(""), fmt.Errorf("%T has no (non-nil) Id field", i)
	}

	iface := f.Interface()
	if id, ok := iface.(bson.ObjectId); ok {
		return id, nil
	}

	if id, ok := iface.(bson.Getter); ok {
		objId, err := id.GetBSON()
		if err != nil {
			return bson.ObjectId(""), err
		}
		if oid, ok := objId.(bson.ObjectId); ok {
			return oid, nil
		}
		return bson.ObjectId(""), fmt.Errorf("Id field of %T did not marshal to a bson.ObjectId", i)
	}

	return bson.ObjectId(""), fmt.Errorf("Unknown type in Id field. Expected mongo.Id or bson.ObjectId. Received: %T", iface)
}

func isPtr(i any) bool {
	return reflect.ValueOf(i).Kind() == reflect.Ptr
}

func typeName(i any) string {
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

func addNewFields(i any) error {
	err := addId(i)
	if err != nil {
		return err
	}

	if err := addCurrentDateTime(i, "CreatedAt"); err != nil {
		return err
	}

	return addCurrentDateTime(i, "UpdatedAt")
}

func addCurrentDateTime(i any, name string) error {
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

func hasStructField(i any, field string) bool {
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

func addId(i any) error {
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

	// Only generate an Id if the caller didn't supply one.
	if f.Kind() == reflect.String && f.Len() == 0 {
		id := f.Interface()
		if _, ok := id.(bson.ObjectId); ok {
			f.Set(reflect.ValueOf(bson.NewObjectId()))
		} else {
			f.SetString(bson.NewObjectId().Hex())
		}
	}

	return nil
}

// Id let's you use a string data type for your models instead of the native
// bson.ObjectId. The main benefit is when you frequently want a hex
// represenation such as for use in web apps. You still need to provide
// the `bson:"_id"` tag.
type Id string

func (i Id) GetBSON() (any, error) {
	if !bson.IsObjectIdHex(string(i)) {
		return nil, fmt.Errorf("invalid ObjectId hex: %q", string(i))
	}
	return bson.ObjectIdHex(string(i)), nil
}

func (i *Id) SetBSON(raw bson.Raw) error {
	var objId bson.ObjectId
	err := raw.Unmarshal(&objId)
	if err != nil {
		return err
	}
	*i = Id(objId.Hex())
	return nil
}
