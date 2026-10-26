package mongo

import (
	"reflect"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
)

type withStringID struct {
	Id        Id `bson:"_id"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type badTime struct {
	Id        bson.ObjectId `bson:"_id"`
	CreatedAt string
}

type noID struct {
	Name string
}

func TestTypeName(t *testing.T) {
	tests := []struct {
		in   any
		want string
	}{
		{MongoTest{}, "MongoTest"},
		{&MongoTest{}, "MongoTest"},
		{[]MongoTest{}, "MongoTest"},
		{&[]MongoTest{}, "MongoTest"},
		{&[]*MongoTest{}, "MongoTest"},
	}
	for _, tt := range tests {
		if got := typeName(tt.in); got != tt.want {
			t.Errorf("typeName(%T) = %q; want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsPtrIsSlice(t *testing.T) {
	if isPtr(MongoTest{}) || !isPtr(&MongoTest{}) {
		t.Error("isPtr wrong")
	}
	var s []MongoTest
	if isSlice(typeOf(MongoTest{})) || !isSlice(typeOf(s)) || !isSlice(typeOf(&s)) {
		t.Error("isSlice wrong")
	}
}

func TestHasStructField(t *testing.T) {
	if !hasStructField(&MongoTest{}, "CreatedAt") {
		t.Error("expected CreatedAt to be found")
	}
	if hasStructField(&MongoTest{}, "Missing") {
		t.Error("did not expect Missing to be found")
	}
	if hasStructField(&[]MongoTest{}, "CreatedAt") {
		t.Error("slices have no struct fields")
	}
}

func TestAddNewFields(t *testing.T) {
	before := time.Now()

	m := &MongoTest{Name: "x"}
	if err := addNewFields(m); err != nil {
		t.Fatal(err)
	}
	if !m.Id.Valid() {
		t.Errorf("ObjectId not set: %q", m.Id)
	}
	if m.CreatedAt.Before(before) || m.UpdatedAt.Before(before) {
		t.Errorf("timestamps not set: %v %v", m.CreatedAt, m.UpdatedAt)
	}

	s := &withStringID{}
	if err := addNewFields(s); err != nil {
		t.Fatal(err)
	}
	if !bson.IsObjectIdHex(string(s.Id)) {
		t.Errorf("string Id not set to an ObjectId hex: %q", s.Id)
	}

	// Structs without Id/CreatedAt/UpdatedAt are left alone.
	if err := addNewFields(&noID{}); err != nil {
		t.Errorf("unexpected error for struct without Id: %v", err)
	}
}

func TestAddNewFieldsErrors(t *testing.T) {
	if err := addNewFields(&badTime{}); err == nil {
		t.Error("expected error for non-time.Time CreatedAt")
	}
	n := 1
	if err := addNewFields(&n); err == nil {
		t.Error("expected error for non-struct record")
	}
}

func TestGetObjIdFromStruct(t *testing.T) {
	oid := bson.NewObjectId()
	got, err := getObjIdFromStruct(&MongoTest{Id: oid})
	if err != nil || got != oid {
		t.Errorf("ObjectId field: got %q, %v", got, err)
	}

	got, err = getObjIdFromStruct(&withStringID{Id: Id(oid.Hex())})
	if err != nil || got != oid {
		t.Errorf("Id field: got %q, %v", got, err)
	}

	if _, err := getObjIdFromStruct(&[]MongoTest{}); err == nil {
		t.Error("expected error for non-struct")
	}
}

func TestIdBSONRoundTrip(t *testing.T) {
	oid := bson.NewObjectId()
	in := withStringID{Id: Id(oid.Hex())}

	data, err := bson.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}

	// On the wire the Id must be stored as a real ObjectId.
	var raw struct {
		Id bson.ObjectId `bson:"_id"`
	}
	if err := bson.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.Id != oid {
		t.Errorf("stored _id = %q; want ObjectId %q", raw.Id, oid)
	}

	var out withStringID
	if err := bson.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Id != in.Id {
		t.Errorf("round trip Id = %q; want %q", out.Id, in.Id)
	}
}

func TestRequiresPointer(t *testing.T) {
	if err := Insert(MongoTest{}); err != NoPtr {
		t.Errorf("Insert: got %v; want NoPtr", err)
	}
	if err := Find(MongoTest{}, nil); err != NoPtr {
		t.Errorf("Find: got %v; want NoPtr", err)
	}
	if err := Update(MongoTest{}); err != NoPtr {
		t.Errorf("Update: got %v; want NoPtr", err)
	}
	if err := Delete(MongoTest{}); err != NoPtr {
		t.Errorf("Delete: got %v; want NoPtr", err)
	}
}

func typeOf(i any) reflect.Type { return reflect.TypeOf(i) }

func TestAddNewFieldsKeepsExistingId(t *testing.T) {
	oid := bson.NewObjectId()
	m := &MongoTest{Id: oid}
	if err := addNewFields(m); err != nil {
		t.Fatal(err)
	}
	if m.Id != oid {
		t.Errorf("caller-supplied ObjectId was replaced: %q -> %q", oid, m.Id)
	}

	s := &withStringID{Id: Id(oid.Hex())}
	if err := addNewFields(s); err != nil {
		t.Fatal(err)
	}
	if s.Id != Id(oid.Hex()) {
		t.Errorf("caller-supplied Id was replaced: %q -> %q", oid.Hex(), s.Id)
	}
}

func TestInvalidIdsReturnErrors(t *testing.T) {
	// None of these may panic.
	if err := FindById(&MongoTest{}, "not-hex"); err == nil {
		t.Error("FindById: expected error for invalid hex")
	}
	if _, err := Id("not-hex").GetBSON(); err == nil {
		t.Error("Id.GetBSON: expected error for invalid hex")
	}
	if _, err := bson.Marshal(withStringID{}); err == nil {
		t.Error("marshaling an empty Id: expected error")
	}
	if _, err := getObjIdFromStruct(&noID{}); err == nil {
		t.Error("getObjIdFromStruct: expected error for struct without Id")
	}
	type ptrID struct{ Id *bson.ObjectId }
	if _, err := getObjIdFromStruct(&ptrID{}); err == nil {
		t.Error("getObjIdFromStruct: expected error for nil Id pointer")
	}
	type intID struct{ Id int }
	if _, err := getObjIdFromStruct(&intID{}); err == nil {
		t.Error("getObjIdFromStruct: expected error for int Id")
	}
}

func TestSetServersRemembersServers(t *testing.T) {
	mu.Lock()
	oldSession, oldServers, oldDB := mgoSession, servers, database
	mgoSession = nil
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		mgoSession, servers, database = oldSession, oldServers, oldDB
		mu.Unlock()
	})

	// mgo rejects unknown URL options before dialing, so this fails fast.
	_ = SetServers("mongodb://localhost/?nosuchoption=1", "db1")
	mu.Lock()
	defer mu.Unlock()
	if servers != "mongodb://localhost/?nosuchoption=1" || database != "db1" {
		t.Errorf("SetServers did not record its arguments: servers=%q database=%q", servers, database)
	}
}
