package dbm

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ytsiuryn/ds-audiodbm/src/entity"
	md "github.com/ytsiuryn/ds-audiomd"
	srv "github.com/ytsiuryn/ds-microservice"
)

func TestDbmService(t *testing.T) {
	// setup code
	startTestService()
	cl := srv.NewRPCClient()

	req := NewAudioDBRequest("delete_entry", &entity.AlbumEntry{Path: "test"})
	requestAnswer(t, cl, req)

	data, err := ioutil.ReadFile("../testdata/test_assumption.json")
	require.NoError(t, err)
	testAssumption := md.NewAssumption(nil)
	require.NoError(t, json.Unmarshal(data, &testAssumption))

	// tests

	t.Run("BaseCommands", func(t *testing.T) {
		correlationID, data, err := srv.CreateCmdRequest("ping")
		require.NoError(t, err)
		cl.Request(ServiceName, correlationID, data)
		respData := cl.Result(correlationID)
		assert.Empty(t, respData)

		correlationID, data, err = srv.CreateCmdRequest("x")
		require.NoError(t, err)
		cl.Request(ServiceName, correlationID, data)
		resp, err := srv.ParseErrorAnswer(cl.Result(correlationID))
		require.NoError(t, err)
		// {"error": "Unknown command: x", "context": "Message dispatcher"}
		assert.NotEmpty(t, resp.Error)
	})

	t.Run("CreateEntry", func(t *testing.T) {
		req.Cmd = "set_entry"
		req.Entry.Status = "without_mandatory_tags"
		req.ImportAssumption(testAssumption)
		answ := requestAnswer(t, cl, req)
		assert.NotZero(t, answ.Entry.ID)
		req.Entry.ID = answ.Entry.ID
	})

	t.Run("GetEntry", func(t *testing.T) {
		req.Cmd = "get_entry"
		req.ClearMetaData()
		answ := requestAnswer(t, cl, req)
		assert.Equal(t, answ.Entry.Path, "test")
		assert.Len(t, answ.Actors, 1)
		assert.Len(t, answ.Pictures, 1)
	})

	t.Run("ChangeEntry", func(t *testing.T) {
		testAssumption.Release.Title = "Changed Title" // изменения  в тестовом образце
		req.Cmd = "set_entry"
		req.ImportAssumption(testAssumption)
		answ := requestAnswer(t, cl, req)
		require.Equal(t, req.Entry.ID, answ.Entry.ID)
		req.Cmd = "get_entry"
		req.ClearMetaData()
		answ = requestAnswer(t, cl, req)
		release := md.NewRelease()
		require.NoError(t, json.Unmarshal(answ.Entry.Json, release))
		assert.Equal(t, release.Title, "Changed Title")
	})

	t.Run("FinalyzeEntry", func(t *testing.T) {
		req.Cmd = "finalyze_entry"
		req.ClearMetaData()
		answ := requestAnswer(t, cl, req)
		assert.Equal(t, answ.Entry.Status, "finalyzed")
	})

	t.Run("DeleteEntry", func(t *testing.T) {
		req.Cmd = "delete_entry"
		answ := requestAnswer(t, cl, req)
		assert.Equal(t, answ, req)
	})

	// tear-down code
	cl.Close()
}

func requestAnswer(t *testing.T, cl *srv.RPCClient, req *AudioDBRequest) *AudioDBRequest {
	corrID, json, err := req.Create()
	require.NoError(t, err)
	cl.Request(ServiceName, corrID, json)

	answ, err := ParseAnswer(cl.Result(corrID))
	require.NoError(t, err)
	return answ.Unwrap()
}

func startTestService() {
	testService := New(os.Getenv("DS_DB_URL"))
	msgs := testService.ConnectToMessageBroker("amqp://guest:guest@localhost:5672/")
	testService.Log.SetLevel(log.DebugLevel)
	go testService.Start(msgs)
}
