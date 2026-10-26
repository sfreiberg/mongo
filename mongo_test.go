package mongo

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

var (
	testObj = &MongoTest{
		Id:   bson.NewObjectId(),
		Name: "testing",
	}

	testObjNoId = &MongoTest{
		Name: "testing no id",
	}
)

type MongoTest struct {
	Id        bson.ObjectId `bson:"_id"`
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	connectOnce sync.Once
	connectErr  error
)

// requireMongo connects to the server in $MONGO_TEST_URL (default
// "localhost") using the "test" database, or skips the test if no
// server that mgo can talk to is reachable. Note that mgo only speaks the
// legacy wire protocol, which MongoDB removed in 5.1/6.0.
func requireMongo(t *testing.T) {
	t.Helper()
	connectOnce.Do(func() {
		url := os.Getenv("MONGO_TEST_URL")
		if url == "" {
			url = "localhost"
		}
		var s *mgo.Session
		s, connectErr = mgo.DialWithTimeout(url, 3*time.Second)
		if connectErr == nil {
			mgoSession, servers, database = s, url, "test"
		}
	})
	if connectErr != nil {
		t.Skipf("no usable MongoDB server (set MONGO_TEST_URL; MongoDB >= 6.0 is not supported by mgo): %v", connectErr)
	}
}

func TestInsert(t *testing.T) {
	requireMongo(t)
	if err := Insert(testObj); err != nil {
		t.Fatal("Couldn't insert record:", err)
	}
}

func TestInsertWithoutPtr(t *testing.T) {
	if err := Insert(*testObj); err != NoPtr {
		t.Fatal("Didn't receive the NoPtr error. Got this instead", err)
	}
}

func TestInsertWithoutId(t *testing.T) {
	requireMongo(t)
	err := Insert(testObjNoId)
	if err != nil {
		t.Fatal("Wasn't able to insert a record without an Id:", err)
	}

	if !testObjNoId.Id.Valid() {
		t.Fatal("Didn't receive a valid id:", testObjNoId.Id.Hex())
	}
}

func TestFind(t *testing.T) {
	requireMongo(t)
	m := &MongoTest{}
	q := bson.M{"_id": testObj.Id}
	err := Find(m, q)
	if err != nil {
		t.Fatal("Couldn't find record. Received the following error:", err)
	}

	if m.Name != "testing" {
		t.Fatal("Couldn't find a record saved earlier.")
	}
}

func TestUpdate(t *testing.T) {
	requireMongo(t)
	testObj.Name = "testing update"
	if err := Update(testObj); err != nil {
		t.Fatal("Couldn't update a record saved earlier:", err)
	}
}

func TestDelete(t *testing.T) {
	requireMongo(t)
	if err := Delete(testObj); err != nil {
		t.Fatal("Couldn't delete record saved earlier:", err)
	}

	if err := Delete(testObjNoId); err != nil {
		t.Fatal("Couldn't delete record saved earlier:", err)
	}
}
