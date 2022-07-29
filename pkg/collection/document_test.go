/*
Copyright © 2020 FairOS Authors
push
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collection_test

import (
	"encoding/json"
	"errors"
	"io"
	"testing"

	f "github.com/fairdatasociety/fairOS-dfs/pkg/file"

	"github.com/fairdatasociety/fairOS-dfs/pkg/account"
	"github.com/fairdatasociety/fairOS-dfs/pkg/blockstore/bee/mock"
	"github.com/fairdatasociety/fairOS-dfs/pkg/collection"
	"github.com/fairdatasociety/fairOS-dfs/pkg/feed"
	"github.com/fairdatasociety/fairOS-dfs/pkg/logging"
)

type TestDocument struct {
	ID        string            `json:"id"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	Age       float64           `json:"age"`
	TagMap    map[string]string `json:"tag_map"`
	TagList   []string          `json:"tag_list"`
}

func TestDocumentStore(t *testing.T) {
	mockClient := mock.NewMockBeeClient()
	logger := logging.New(io.Discard, 0)
	acc := account.New(logger)
	ai := acc.GetUserAccountInfo()
	_, _, err := acc.CreateUserAccount("password", "")
	if err != nil {
		t.Fatal(err)
	}
	fd := feed.New(acc.GetUserAccountInfo(), mockClient, logger)
	user := acc.GetAddress(account.UserAccountIndex)
	file := f.NewFile("pod1", mockClient, fd, user, logger)
	docStore := collection.NewDocumentStore("pod1", fd, ai, user, file, mockClient, logger)

	t.Run("create_document_db", func(t *testing.T) {
		// create a document DB
		createDocumentDBs(t, []string{"docdb_0"}, docStore, nil)

		// load the schem and check the count of simple indexes
		schema := loadSchemaAndCheckSimpleIndexCount(t, docStore, "docdb_0", 1)

		// check the default index
		checkIndex(t, schema.SimpleIndexes[0], collection.DefaultIndexFieldName, collection.StringIndex)
	})

	t.Run("delete_document_db", func(t *testing.T) {
		// create multiple document DB
		createDocumentDBs(t, []string{"docdb_1_1", "docdb_1_2", "docdb_1_3"}, docStore, nil)
		checkIfDBsExists(t, []string{"docdb_1_1", "docdb_1_2", "docdb_1_3"}, docStore)

		// delete the db in the middle
		err = docStore.DeleteDocumentDB("docdb_1_2")
		if err != nil {
			t.Fatal(err)
		}

		// check if other two db exists
		checkIfDBsExists(t, []string{"docdb_1_1", "docdb_1_3"}, docStore)
	})

	t.Run("create_document_db_with_multiple_indexes", func(t *testing.T) {
		// create a document DB and add simple indexes
		si := make(map[string]collection.IndexType)
		si["field1"] = collection.StringIndex
		si["field2"] = collection.NumberIndex
		si["field3"] = collection.MapIndex
		si["field4"] = collection.ListIndex
		createDocumentDBs(t, []string{"docdb_2"}, docStore, si)

		// load the schem and check the count of simple indexes
		schema := loadSchemaAndCheckSimpleIndexCount(t, docStore, "docdb_2", 3)

		// first check the default index
		checkIndex(t, schema.SimpleIndexes[0], collection.DefaultIndexFieldName, collection.StringIndex)

		checkIndex(t, schema.SimpleIndexes[0], "id", collection.StringIndex)

		//second check the field in index 1
		if schema.SimpleIndexes[1].FieldName == "field1" {
			checkIndex(t, schema.SimpleIndexes[1], "field1", collection.StringIndex)
		} else {
			checkIndex(t, schema.SimpleIndexes[1], "field2", collection.NumberIndex)
		}

		//third check the field in index 2
		if schema.SimpleIndexes[2].FieldName == "field2" {
			checkIndex(t, schema.SimpleIndexes[2], "field2", collection.NumberIndex)
		} else {
			checkIndex(t, schema.SimpleIndexes[2], "field1", collection.StringIndex)
		}

		if schema.MapIndexes[0].FieldName == "field3." {
			checkIndex(t, schema.MapIndexes[0], "field3.", collection.MapIndex)
		}

		if schema.ListIndexes[0].FieldName == "field4" {
			checkIndex(t, schema.ListIndexes[0], "field4", collection.ListIndex)
		}

	})

	t.Run("create_and open_document_db", func(t *testing.T) {
		// create a document DB
		createDocumentDBs(t, []string{"docdb_3"}, docStore, nil)

		err := docStore.OpenDocumentDB("docdb_3")
		if err != nil {
			t.Fatal(err)
		}

		// check if the DB is opened properly
		if !docStore.IsDBOpened("docdb_3") {
			t.Fatalf("db not opened")
		}

	})

	t.Run("put_and_get", func(t *testing.T) {
		// create a document DB
		createDocumentDBs(t, []string{"docdb_4"}, docStore, nil)

		err := docStore.OpenDocumentDB("docdb_4")
		if err != nil {
			t.Fatal(err)
		}

		// create a json document
		document1 := &TestDocument{
			ID:        "1",
			FirstName: "John",
			LastName:  "Doe",
			Age:       25,
		}
		data, err := json.Marshal(document1)
		if err != nil {
			t.Fatal(err)
		}

		// insert the docment in the DB
		err = docStore.Put("docdb_4", data)
		if err != nil {
			t.Fatal(err)
		}

		// get the data and test if the retreived data is okay
		gotData, err := docStore.Get("docdb_4", "1")
		if err != nil {
			t.Fatal(err)
		}

		var doc TestDocument
		err = json.Unmarshal(gotData, &doc)
		if err != nil {
			t.Fatal(err)
		}
		if doc.ID != document1.ID ||
			doc.FirstName != document1.FirstName ||
			doc.LastName != document1.LastName ||
			doc.Age != document1.Age {
			t.Fatalf("invalid json data received")
		}
	})

	t.Run("put_and_get_multiple_index", func(t *testing.T) {
		// create a document DB
		si := make(map[string]collection.IndexType)
		si["first_name"] = collection.StringIndex
		si["age"] = collection.NumberIndex
		si["tag_map"] = collection.MapIndex
		si["tag_list"] = collection.ListIndex
		createDocumentDBs(t, []string{"docdb_5"}, docStore, si)

		err := docStore.OpenDocumentDB("docdb_5")
		if err != nil {
			t.Fatal(err)
		}

		// Add documents
		createTestDocuments(t, docStore, "docdb_5")

		// get string index and check if the documents returned are okay
		docs, err := docStore.Get("docdb_5", "2")
		if err != nil {
			t.Fatal(err)
		}
		var gotDoc TestDocument
		err = json.Unmarshal(docs, &gotDoc)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc.ID != "2" ||
			gotDoc.FirstName != "John" ||
			gotDoc.LastName != "boy" ||
			gotDoc.Age != 25 ||
			gotDoc.TagMap["tgf21"] != "tgv21" ||
			gotDoc.TagMap["tgf22"] != "tgv22" {
			t.Fatalf("invalid json data received")
		}
	})

	t.Run("count_all", func(t *testing.T) {
		// create a document DB
		si := make(map[string]collection.IndexType)
		si["first_name"] = collection.StringIndex
		si["age"] = collection.NumberIndex
		si["tag_map"] = collection.MapIndex
		si["tag_list"] = collection.ListIndex
		createDocumentDBs(t, []string{"docdb_6"}, docStore, si)

		err := docStore.OpenDocumentDB("docdb_6")
		if err != nil {
			t.Fatal(err)
		}

		// Add documents
		createTestDocuments(t, docStore, "docdb_6")

		count1, err := docStore.Count("docdb_6", "")
		if err != nil {
			t.Fatal(err)
		}

		if count1 != 5 {
			t.Fatalf("expected count %d, got %d", 5, count1)
		}

	})

	t.Run("count_with_expr", func(t *testing.T) {
		// create a document DB
		si := make(map[string]collection.IndexType)
		si["first_name"] = collection.StringIndex
		si["age"] = collection.NumberIndex
		si["tag_map"] = collection.MapIndex
		si["tag_list"] = collection.ListIndex
		createDocumentDBs(t, []string{"docdb_7"}, docStore, si)

		err := docStore.OpenDocumentDB("docdb_7")
		if err != nil {
			t.Fatal(err)
		}

		// Add documents
		createTestDocuments(t, docStore, "docdb_7")

		// String count
		count1, err := docStore.Count("docdb_7", "first_name=John")
		if err != nil {
			t.Fatal(err)
		}
		if count1 != 2 {
			t.Fatalf("expected count %d, got %d", 2, count1)
		}

		count1, err = docStore.Count("docdb_7", "tag_map=tgf11:tgv11")
		if err != nil {
			t.Fatal(err)
		}
		if count1 != 1 {
			t.Fatalf("expected count %d, got %d", 1, count1)
		}

		// Number =
		count2, err := docStore.Count("docdb_7", "age=25")
		if err != nil {
			t.Fatal(err)
		}
		if count2 != 3 {
			t.Fatalf("expected count %d, got %d", 3, count2)
		}

		// Number =>
		count3, err := docStore.Count("docdb_7", "age=>30")
		if err != nil {
			t.Fatal(err)
		}
		if count3 != 2 {
			t.Fatalf("expected count %d, got %d", 2, count3)
		}

		// Number >
		count4, err := docStore.Count("docdb_7", "age>30")
		if err != nil {
			t.Fatal(err)
		}
		if count4 != 1 {
			t.Fatalf("expected count %d, got %d", 1, count4)
		}
	})

	t.Run("find", func(t *testing.T) {
		// create a document DB
		si := make(map[string]collection.IndexType)
		si["first_name"] = collection.StringIndex
		si["age"] = collection.NumberIndex
		si["tag_map"] = collection.MapIndex
		si["tag_list"] = collection.ListIndex
		createDocumentDBs(t, []string{"docdb_8"}, docStore, si)

		err := docStore.OpenDocumentDB("docdb_8")
		if err != nil {
			t.Fatal(err)
		}

		// Add documents
		createTestDocuments(t, docStore, "docdb_8")

		// String =
		docs, err := docStore.Find("docdb_8", "first_name=John", -1)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 2 {
			t.Fatalf("expected count %d, got %d", 2, len(docs))
		}
		var gotDoc1 TestDocument
		err = json.Unmarshal(docs[0], &gotDoc1)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc1.ID != "1" ||
			gotDoc1.FirstName != "John" ||
			gotDoc1.LastName != "Doe" ||
			gotDoc1.Age != 45.523793600000005 {
			t.Fatalf("invalid json data received")
		}
		var gotDoc2 TestDocument
		err = json.Unmarshal(docs[1], &gotDoc2)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc2.ID != "2" ||
			gotDoc2.FirstName != "John" ||
			gotDoc2.LastName != "boy" ||
			gotDoc2.Age != 25 {
			t.Fatalf("invalid json data received")
		}

		// tag
		docs, err = docStore.Find("docdb_8", "tag_map=tgf21:tgv21", -1)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected count %d, got %d", 1, len(docs))
		}
		err = json.Unmarshal(docs[0], &gotDoc2)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc2.ID != "2" ||
			gotDoc2.FirstName != "John" ||
			gotDoc2.LastName != "boy" ||
			gotDoc2.Age != 25 ||
			gotDoc2.TagMap["tgf21"] != "tgv21" {
			t.Fatalf("invalid json data received")
		}

		// Number =
		docs, err = docStore.Find("docdb_8", "age=25", -1)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 3 {
			t.Fatalf("expected count %d, got %d", 3, len(docs))
		}
		err = json.Unmarshal(docs[0], &gotDoc1)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc1.ID != "2" ||
			gotDoc1.FirstName != "John" ||
			gotDoc1.LastName != "boy" ||
			gotDoc1.Age != 25 {
			t.Fatalf("invalid json data received")
		}
		err = json.Unmarshal(docs[1], &gotDoc2)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc2.ID != "4" ||
			gotDoc2.FirstName != "Charlie" ||
			gotDoc2.LastName != "chaplin" ||
			gotDoc2.Age != 25 {
			t.Fatalf("invalid json data received")
		}
		var gotDoc3 TestDocument
		err = json.Unmarshal(docs[2], &gotDoc3)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc3.ID != "5" ||
			gotDoc3.FirstName != "Alice" ||
			gotDoc3.LastName != "wonderland" ||
			gotDoc3.Age != 25 {
			t.Fatalf("invalid json data received")
		}

		// Number = with limit
		docs, err = docStore.Find("docdb_8", "age=25", 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 2 {
			t.Fatalf("expected count %d, got %d", 2, len(docs))
		}
		err = json.Unmarshal(docs[0], &gotDoc1)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc1.ID != "2" ||
			gotDoc1.FirstName != "John" ||
			gotDoc1.LastName != "boy" ||
			gotDoc1.Age != 25 {
			t.Fatalf("invalid json data received")
		}
		err = json.Unmarshal(docs[1], &gotDoc2)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc2.ID != "4" ||
			gotDoc2.FirstName != "Charlie" ||
			gotDoc2.LastName != "chaplin" ||
			gotDoc2.Age != 25 {
			t.Fatalf("invalid json data received")
		}

		// Number =>
		docs, err = docStore.Find("docdb_8", "age=>30", -1)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 2 {
			t.Fatalf("expected count %d, got %d", 2, len(docs))
		}
		err = json.Unmarshal(docs[0], &gotDoc1)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc1.ID != "3" ||
			gotDoc1.FirstName != "Bob" ||
			gotDoc1.LastName != "michel" ||
			gotDoc1.Age != 30 {
			t.Fatalf("invalid json data received")
		}
		err = json.Unmarshal(docs[1], &gotDoc2)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc2.ID != "1" ||
			gotDoc2.FirstName != "John" ||
			gotDoc2.LastName != "Doe" ||
			gotDoc2.Age != 45.523793600000005 {
			t.Fatalf("invalid json data received")
		}

		// Number >
		docs, err = docStore.Find("docdb_8", "age>30", -1)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected count %d, got %d", 1, len(docs))
		}
		err = json.Unmarshal(docs[0], &gotDoc1)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc1.ID != "1" ||
			gotDoc1.FirstName != "John" ||
			gotDoc1.LastName != "Doe" ||
			gotDoc1.Age != 45.523793600000005 {
			t.Fatalf("invalid json data received")
		}
	})

	t.Run("del", func(t *testing.T) {
		// create a document DB
		si := make(map[string]collection.IndexType)
		si["first_name"] = collection.StringIndex
		si["age"] = collection.NumberIndex
		createDocumentDBs(t, []string{"docdb_9"}, docStore, si)

		err := docStore.OpenDocumentDB("docdb_9")
		if err != nil {
			t.Fatal(err)
		}

		// Add document and get to see if it is added
		tag1 := make(map[string]string)
		tag1["tgf11"] = "tgv11"
		tag1["tgf12"] = "tgv12"
		var list1 []string
		list1 = append(list1, "lst11", "lst12")
		addDocument(t, docStore, "docdb_9", "1", "John", "Doe", 45, tag1, list1)
		docs, err := docStore.Get("docdb_9", "1")
		if err != nil {
			t.Fatal(err)
		}
		var gotDoc TestDocument
		err = json.Unmarshal(docs, &gotDoc)
		if err != nil {
			t.Fatal(err)
		}
		if gotDoc.ID != "1" ||
			gotDoc.FirstName != "John" ||
			gotDoc.LastName != "Doe" ||
			gotDoc.Age != 45 {
			t.Fatalf("invalid json data received")
		}

		// del document
		err = docStore.Del("docdb_9", "1")
		if err != nil {
			t.Fatal(err)
		}
		_, err = docStore.Get("docdb_9", "1")
		if !errors.Is(err, collection.ErrEntryNotFound) {
			t.Fatal(err)
		}
	})

	t.Run("add_add", func(t *testing.T) {
		// create a document DB
		si := make(map[string]collection.IndexType)
		si["first_name"] = collection.StringIndex
		si["age"] = collection.NumberIndex
		createDocumentDBs(t, []string{"docdb_10"}, docStore, si)

		err := docStore.OpenDocumentDB("docdb_10")
		if err != nil {
			t.Fatal(err)
		}

		tag1 := make(map[string]string)
		tag1["tgf11"] = "tgv11"
		tag1["tgf12"] = "tgv12"
		var list1 []string
		list1 = append(list1, "lst11", "lst12")
		addDocument(t, docStore, "docdb_10", "1", "John", "Doe", 45, tag1, list1)
		addDocument(t, docStore, "docdb_10", "1", "John", "Doe", 25, tag1, list1)

		// count the total docs using id field
		count1, err := docStore.Count("docdb_10", "")
		if err != nil {
			t.Fatal(err)
		}
		if count1 != 1 {
			t.Fatalf("expected count %d, got %d", 1, count1)
		}

		// count the total docs using another index to make sure we dont have it any index
		docs, err := docStore.Find("docdb_10", "age=>20", -1)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected count %d, got %d", 1, len(docs))
		}
	})

	t.Run("batch-mutable", func(t *testing.T) {
		// create a document DB
		si := make(map[string]collection.IndexType)
		si["first_name"] = collection.StringIndex
		si["age"] = collection.NumberIndex
		si["tag_map"] = collection.MapIndex
		si["tag_list"] = collection.ListIndex
		createDocumentDBs(t, []string{"docdb_11"}, docStore, si)

		err := docStore.OpenDocumentDB("docdb_11")
		if err != nil {
			t.Fatal(err)
		}

		docBatch, err := docStore.CreateDocBatch("docdb_11")
		if err != nil {
			t.Fatal(err)
		}

		tag1 := make(map[string]string)
		tag1["tgf11"] = "tgv11"
		tag1["tgf12"] = "tgv12"
		var list1 []string
		list1 = append(list1, "lst11", "lst12")
		addBatchDocument(t, docStore, docBatch, "1", "John", "Doe", 45, tag1, list1)
		tag2 := make(map[string]string)
		tag2["tgf21"] = "tgv21"
		tag2["tgf22"] = "tgv22"
		var list2 []string
		list2 = append(list2, "lst21", "lst22")
		addBatchDocument(t, docStore, docBatch, "2", "John", "boy", 25, tag2, list2)
		tag3 := make(map[string]string)
		tag3["tgf31"] = "tgv31"
		tag3["tgf32"] = "tgv32"
		var list3 []string
		list3 = append(list3, "lst31", "lst32")
		addBatchDocument(t, docStore, docBatch, "3", "Alice", "wonderland", 20, tag3, list3)
		tag4 := make(map[string]string)
		tag4["tgf41"] = "tgv41"
		tag4["tgf42"] = "tgv42"
		var list4 []string
		list4 = append(list4, "lst41", "lst42")
		addBatchDocument(t, docStore, docBatch, "4", "John", "Doe", 35, tag4, list4) // this tests the overwriting in batch

		err = docStore.DocBatchWrite(docBatch, "")
		if err != nil {
			t.Fatal(err)
		}

		// count the total docs using id field
		count1, err := docStore.Count("docdb_11", "")
		if err != nil {
			t.Fatal(err)
		}
		if count1 != 4 {
			t.Fatalf("expected count %d, got %d", 4, count1)
		}

		// count the total docs using another index to make sure we dont have it any index
		docs, err := docStore.Find("docdb_11", "age=>20", -1)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 4 {
			t.Fatalf("expected count %d, got %d", 3, len(docs))
		}

		// tag
		docs, err = docStore.Find("docdb_11", "tag_map=tgf21:tgv21", -1)
		if err != nil {
			t.Fatal(err)
		}
		if len(docs) != 1 {
			t.Fatalf("expected count %d, got %d", 1, len(docs))
		}
		err = docStore.DeleteDocumentDB("docdb_11")
		if err != nil {
			t.Fatal(err)
		}
	})

	//t.Run("batch-immutable", func(t *testing.T) {
	//	// create a document DB
	//	si := make(map[string]collection.IndexType)
	//	si["first_name"] = collection.StringIndex
	//	si["age"] = collection.NumberIndex
	//	si["tag_map"] = collection.MapIndex
	//	si["tag_list"] = collection.ListIndex
	//	//createDocumentDBs(t, []string{"docdb_12"}, docStore, si)
	//	err := docStore.CreateDocumentDB("docdb_12", si, false)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	err = docStore.OpenDocumentDB("docdb_12")
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	docBatch, err := docStore.CreateDocBatch("docdb_12")
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	tag1 := make(map[string]string)
	//	tag1["tgf11"] = "tgv11"
	//	tag1["tgf12"] = "tgv12"
	//	var list1 []string
	//	list1 = append(list1, "lst11")
	//	list1 = append(list1, "lst12")
	//	addBatchDocument(t, docStore, docBatch, "1", "John", "Doe", 45, tag1, list1)
	//	tag2 := make(map[string]string)
	//	tag2["tgf21"] = "tgv21"
	//	tag2["tgf22"] = "tgv22"
	//	var list2 []string
	//	list2 = append(list2, "lst21")
	//	list2 = append(list2, "lst22")
	//	addBatchDocument(t, docStore, docBatch, "2", "John", "boy", 25, tag2, list2)
	//	tag3 := make(map[string]string)
	//	tag3["tgf31"] = "tgv31"
	//	tag3["tgf32"] = "tgv32"
	//	var list3 []string
	//	list3 = append(list3, "lst31")
	//	list3 = append(list3, "lst32")
	//	addBatchDocument(t, docStore, docBatch, "3", "Alice", "wonderland", 20, tag3, list3)
	//	tag4 := make(map[string]string)
	//	tag4["tgf41"] = "tgv41"
	//	tag4["tgf42"] = "tgv42"
	//	var list4 []string
	//	list4 = append(list4, "lst41")
	//	list4 = append(list4, "lst42")
	//	addBatchDocument(t, docStore, docBatch, "4", "John", "Doe", 35, tag4, list4) // this tests the overwriting in batch
	//
	//	err = docStore.DocBatchWrite(docBatch, "")
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	// count the total docs using id field
	//	count1, err := docStore.Count("docdb_12", "")
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	if count1 != 4 {
	//		t.Fatalf("expected count %d, got %d", 4, count1)
	//	}
	//
	//	// count the total docs using another index to make sure we dont have it any index
	//	docs, err := docStore.Find("docdb_12", "age=>20", -1)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	if len(docs) != 4 {
	//		t.Fatalf("expected count %d, got %d", 4, len(docs))
	//	}
	//
	//	// tag
	//	docs, err = docStore.Find("docdb_12", "tag_map=tgf21:tgv21", -1)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	if len(docs) != 1 {
	//		t.Fatalf("expected count %d, got %d", 1, len(docs))
	//	}
	//	err = docStore.DeleteDocumentDB("docdb_12")
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//})

}

func createDocumentDBs(t *testing.T, dbNames []string, docStore *collection.Document, si map[string]collection.IndexType) {
	t.Helper()
	for _, dbName := range dbNames {
		err := docStore.CreateDocumentDB(dbName, si, true)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func checkIfDBsExists(t *testing.T, dbNames []string, docStore *collection.Document) {
	t.Helper()
	tables, err := docStore.LoadDocumentDBSchemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, tableName := range dbNames {
		if _, found := tables[tableName]; !found {
			t.Fatalf("document db not found")
		}
	}
}

func loadSchemaAndCheckSimpleIndexCount(t *testing.T, docStore *collection.Document, dbName string, count int) collection.DBSchema {
	t.Helper()
	tables, err := docStore.LoadDocumentDBSchemas()
	if err != nil {
		t.Fatal(err)
	}
	schema, found := tables[dbName]
	if !found {
		t.Fatalf("document db not found in schema")
	}
	if len(schema.SimpleIndexes) != count {
		t.Fatalf("index count mismatch")
	}
	return schema
}

func checkIndex(t *testing.T, si collection.SIndex, filedName string, idxType collection.IndexType) {
	t.Helper()
	if si.FieldName != filedName {
		t.Fatalf("index field not found: %s, %s", si.FieldName, filedName)
	}
	if si.FieldType != idxType {
		t.Fatalf("index field type is not correct: %s, %s", si.FieldType, idxType)
	}
}

func createTestDocuments(t *testing.T, docStore *collection.Document, dbName string) {
	t.Helper()
	tag1 := make(map[string]string)
	tag1["tgf11"] = "tgv11"
	tag1["tgf12"] = "tgv12"
	var list1 []string
	list1 = append(list1, "lst11", "lst12")
	addDocument(t, docStore, dbName, "1", "John", "Doe", 45.523793600000005, tag1, list1)
	tag2 := make(map[string]string)
	tag2["tgf21"] = "tgv21"
	tag2["tgf22"] = "tgv22"
	var list2 []string
	list2 = append(list2, "lst21", "lst22")
	addDocument(t, docStore, dbName, "2", "John", "boy", 25, tag2, list2)
	tag3 := make(map[string]string)
	tag3["tgf31"] = "tgv31"
	tag3["tgf32"] = "tgv32"
	var list3 []string
	list3 = append(list3, "lst31", "lst32")
	addDocument(t, docStore, dbName, "3", "Bob", "michel", 30, tag3, list3)
	tag4 := make(map[string]string)
	tag4["tgf41"] = "tgv41"
	tag4["tgf42"] = "tgv42"
	var list4 []string
	list4 = append(list4, "lst41", "lst42")
	addDocument(t, docStore, dbName, "4", "Charlie", "chaplin", 25, tag4, list4)
	tag5 := make(map[string]string)
	tag5["tgf51"] = "tgv51"
	tag5["tgf52"] = "tgv52"
	var list5 []string
	list5 = append(list5, "lst51", "lst52")
	addDocument(t, docStore, dbName, "5", "Alice", "wonderland", 25, tag5, list5)
}

func addDocument(t *testing.T, docStore *collection.Document, dbName, id, fname, lname string, age float64, tagMap map[string]string, tagList []string) {
	t.Helper()
	// create the doc
	doc := &TestDocument{
		ID:        id,
		FirstName: fname,
		LastName:  lname,
		Age:       age,
		TagMap:    tagMap,
		TagList:   tagList,
	}

	// marshall the doc
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}

	// insert the docment in the DB
	err = docStore.Put(dbName, data)
	if err != nil {
		t.Fatal(err)
	}
}

func addBatchDocument(t *testing.T, docStore *collection.Document, docBatch *collection.DocBatch, id, fname, lname string, age float64, tagMap map[string]string, tagList []string) {
	t.Helper()
	t.Run("valid-json", func(t *testing.T) {
		// create the doc
		doc := &TestDocument{
			ID:        id,
			FirstName: fname,
			LastName:  lname,
			Age:       age,
			TagMap:    tagMap,
			TagList:   tagList,
		}

		// marshall the doc
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}

		// insert the document in the batch
		err = docStore.DocBatchPut(docBatch, data, 0)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("invalid-json", func(t *testing.T) {
		// create the doc
		doc := TestDocument{
			ID:        id,
			FirstName: fname,
			LastName:  lname,
			Age:       age,
			TagMap:    tagMap,
			TagList:   tagList,
		}

		// marshall the doc
		data, err := json.Marshal([]TestDocument{doc})
		if err != nil {
			t.Fatal(err)
		}

		// insert the document in the batch
		err = docStore.DocBatchPut(docBatch, data, 0)
		if err != collection.ErrUnknownJsonFormat {
			t.Fatal(err)
		}
	})

}
