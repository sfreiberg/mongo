package mongo

import (
	"labix.org/v2/mgo/bson"

	"testing"
	"time"
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

func TestConnection(t *testing.T) {
	if err := SetServers("localhost", "test"); err != nil {
		t.Fatal("Couldn't connect to mongo server at localhost")
	}
}

func TestInsert(t *testing.T) {
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
	err := Insert(testObjNoId)
	if err != nil {
		t.Fatal("Wasn't able to insert a record without an Id:", err)
	}

	if !testObjNoId.Id.Valid() {
		t.Fatal("Didn't receive a valid id:", testObjNoId.Id.Hex())
	}
}

func TestFind(t *testing.T) {
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
	testObj.Name = "testing update"
	if err := Update(testObj); err != nil {
		t.Fatal("Couldn't update a record saved earlier:", err)
	}
}

func TestDelete(t *testing.T) {
	if err := Delete(testObj); err != nil {
		t.Fatal("Couldn't delete record saved earlier:", err)
	}

	if err := Delete(testObjNoId); err != nil {
		t.Fatal("Couldn't delete record saved earlier:", err)
	}
}
