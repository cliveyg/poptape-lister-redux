package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// ---- Fake errorable collection ----
type fakeCollection struct {
	items     map[string]bson.M
	insertErr error
	deleteErr error
	findErr   error
}

func newFakeCollection() *fakeCollection {
	return &fakeCollection{
		items: make(map[string]bson.M),
	}
}

func (c *fakeCollection) InsertOne(ctx context.Context, doc interface{}) error {
	if c.insertErr != nil {
		return c.insertErr
	}
	m := doc.(bson.M)
	id := m["uuid"].(string)
	if _, exists := c.items[id]; exists {
		return errors.New("duplicate")
	}
	c.items[id] = m
	return nil
}
func (c *fakeCollection) DeleteOne(ctx context.Context, filter interface{}) error {
	if c.deleteErr != nil {
		return c.deleteErr
	}
	id := filter.(bson.M)["uuid"].(string)
	if _, exists := c.items[id]; !exists {
		return errors.New("notfound")
	}
	delete(c.items, id)
	return nil
}
func (c *fakeCollection) Find(ctx context.Context, filter interface{}) ([]bson.M, error) {
	if c.findErr != nil {
		return nil, c.findErr
	}
	var results []bson.M
	for _, v := range c.items {
		results = append(results, v)
	}
	return results, nil
}
func (c *fakeCollection) Clear() {
	c.items = make(map[string]bson.M)
}

// ---- Fake App for testing ----

type fakeApp struct {
	*App
	Watchlist  *fakeCollection
	Favourites *fakeCollection
	Viewed     *fakeCollection
	RecentBids *fakeCollection
	Purchased  *fakeCollection
}

func newFakeApp() *fakeApp {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	app := &App{Log: &logger}
	fapp := &fakeApp{
		App:        app,
		Watchlist:  newFakeCollection(),
		Favourites: newFakeCollection(),
		Viewed:     newFakeCollection(),
		RecentBids: newFakeCollection(),
		Purchased:  newFakeCollection(),
	}
	gin.SetMode(gin.TestMode)
	app.Router = gin.Default()
	app.Router.Use(fapp.fakeAuthMiddleware())
	app.initialiseRoutes()
	return fapp
}

func (a *fakeApp) fakeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/list/status" || strings.HasPrefix(c.Request.URL.Path, "/list/watching") {
			c.Next()
			return
		}
		token := c.Request.Header.Get("X-Access-Token")
		if token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set("userID", "testuser")
		c.Next()
	}
}

// ---- Test helpers ----

func makeReq(method, url string, body interface{}, withAuth bool) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	if withAuth {
		req.Header.Set("X-Access-Token", "goodtoken")
	}
	return req
}

func doReq(a *App, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)
	return rr
}

func parseBody(t *testing.T, resp *httptest.ResponseRecorder) map[string]interface{} {
	var out map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &out))
	return out
}

// ---- Shared test logic for all lists ----

type listSpec struct {
	name       string
	url        string
	collection func(a *fakeApp) *fakeCollection
}

var allLists = []listSpec{
	{"watchlist", "/list/watchlist", func(a *fakeApp) *fakeCollection { return a.Watchlist }},
	{"favourites", "/list/favourites", func(a *fakeApp) *fakeCollection { return a.Favourites }},
	{"viewed", "/list/viewed", func(a *fakeApp) *fakeCollection { return a.Viewed }},
	{"recentbids", "/list/recentbids", func(a *fakeApp) *fakeCollection { return a.RecentBids }},
	{"purchased", "/list/purchased", func(a *fakeApp) *fakeCollection { return a.Purchased }},
}

// ---- TESTS ----

func TestStatusRoute(t *testing.T) {
	app := newFakeApp()
	req := makeReq("GET", "/list/status", nil, false)
	resp := doReq(app.App, req)
	require.Equal(t, http.StatusOK, resp.Code)
	out := parseBody(t, resp)
	require.Contains(t, out["message"], "System running")
}

func TestWatchingCount_Unauthenticated(t *testing.T) {
	app := newFakeApp()
	req := makeReq("GET", "/list/watching/xyz", nil, false)
	resp := doReq(app.App, req)
	require.Equal(t, http.StatusOK, resp.Code)
}

func Test404Route(t *testing.T) {
	app := newFakeApp()
	req := makeReq("GET", "/list/not-a-route", nil, false)
	resp := doReq(app.App, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
	out := parseBody(t, resp)
	require.Contains(t, strings.ToLower(out["message"].(string)), "not found")
}

func TestProtectedRoutes_RequireAuth(t *testing.T) {
	app := newFakeApp()
	for _, spec := range allLists {
		req := makeReq("GET", spec.url, nil, false)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusUnauthorized, resp.Code)
	}
}

func TestList_GET_Empty(t *testing.T) {
	for _, spec := range allLists {
		app := newFakeApp()
		spec.collection(app).Clear()
		req := makeReq("GET", spec.url, nil, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusOK, resp.Code)
		out := parseBody(t, resp)
		require.Contains(t, out, spec.name)
	}
}

func TestList_GET_DBError(t *testing.T) {
	for _, spec := range allLists {
		app := newFakeApp()
		spec.collection(app).findErr = errors.New("db error")
		req := makeReq("GET", spec.url, nil, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusInternalServerError, resp.Code)
		out := parseBody(t, resp)
		require.Contains(t, strings.ToLower(out["message"].(string)), "error")
	}
}

func TestList_POST_BadBody(t *testing.T) {
	for _, spec := range allLists {
		app := newFakeApp()
		req, _ := http.NewRequest("POST", spec.url, bytes.NewBufferString("notjson"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Access-Token", "goodtoken")
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusBadRequest, resp.Code)
	}
}

func TestList_POST_MissingUUID(t *testing.T) {
	for _, spec := range allLists {
		app := newFakeApp()
		body := map[string]interface{}{}
		req := makeReq("POST", spec.url, body, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusBadRequest, resp.Code)
	}
}

func TestList_POST_BadUUID(t *testing.T) {
	for _, spec := range allLists {
		app := newFakeApp()
		body := map[string]interface{}{"uuid": "not-a-uuid"}
		req := makeReq("POST", spec.url, body, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusBadRequest, resp.Code)
	}
}

func TestList_POST_Success(t *testing.T) {
	for _, spec := range allLists {
		app := newFakeApp()
		id := uuid.New().String()
		body := map[string]interface{}{"uuid": id}
		req := makeReq("POST", spec.url, body, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusOK, resp.Code)
	}
}

func TestList_POST_Duplicate(t *testing.T) {
	for _, spec := range allLists {
		app := newFakeApp()
		id := uuid.New().String()
		body := map[string]interface{}{"uuid": id}
		_ = spec.collection(app).InsertOne(context.Background(), bson.M{"uuid": id})
		req := makeReq("POST", spec.url, body, true)
		resp := doReq(app.App, req)
		// Assuming duplicate is handled as 409
		require.Equal(t, http.StatusConflict, resp.Code)
	}
}

func TestList_POST_DBError(t *testing.T) {
	for _, spec := range allLists {
		app := newFakeApp()
		spec.collection(app).insertErr = errors.New("db insert error")
		id := uuid.New().String()
		body := map[string]interface{}{"uuid": id}
		req := makeReq("POST", spec.url, body, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusInternalServerError, resp.Code)
		out := parseBody(t, resp)
		require.Contains(t, strings.ToLower(out["message"].(string)), "error")
	}
}

func TestList_DELETE_BadBody(t *testing.T) {
	for _, spec := range []listSpec{allLists[0], allLists[1]} {
		app := newFakeApp()
		req, _ := http.NewRequest("DELETE", spec.url, bytes.NewBufferString("notjson"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Access-Token", "goodtoken")
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusBadRequest, resp.Code)
	}
}

func TestList_DELETE_MissingUUID(t *testing.T) {
	for _, spec := range []listSpec{allLists[0], allLists[1]} {
		app := newFakeApp()
		body := map[string]interface{}{}
		req := makeReq("DELETE", spec.url, body, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusBadRequest, resp.Code)
	}
}

func TestList_DELETE_BadUUID(t *testing.T) {
	for _, spec := range []listSpec{allLists[0], allLists[1]} {
		app := newFakeApp()
		body := map[string]interface{}{"uuid": "not-a-uuid"}
		req := makeReq("DELETE", spec.url, body, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusBadRequest, resp.Code)
	}
}

func TestList_DELETE_Success(t *testing.T) {
	for _, spec := range []listSpec{allLists[0], allLists[1]} {
		app := newFakeApp()
		id := uuid.New().String()
		_ = spec.collection(app).InsertOne(context.Background(), bson.M{"uuid": id})
		body := map[string]interface{}{"uuid": id}
		req := makeReq("DELETE", spec.url, body, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusOK, resp.Code)
	}
}

func TestList_DELETE_NotFound(t *testing.T) {
	for _, spec := range []listSpec{allLists[0], allLists[1]} {
		app := newFakeApp()
		id := uuid.New().String()
		body := map[string]interface{}{"uuid": id}
		req := makeReq("DELETE", spec.url, body, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusNotFound, resp.Code)
	}
}

func TestList_DELETE_DBError(t *testing.T) {
	for _, spec := range []listSpec{allLists[0], allLists[1]} {
		app := newFakeApp()
		spec.collection(app).deleteErr = errors.New("db delete error")
		id := uuid.New().String()
		_ = spec.collection(app).InsertOne(context.Background(), bson.M{"uuid": id})
		body := map[string]interface{}{"uuid": id}
		req := makeReq("DELETE", spec.url, body, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusInternalServerError, resp.Code)
		out := parseBody(t, resp)
		require.Contains(t, strings.ToLower(out["message"].(string)), "error")
	}
}

func TestList_GET_WithData(t *testing.T) {
	for _, spec := range allLists {
		app := newFakeApp()
		id := uuid.New().String()
		_ = spec.collection(app).InsertOne(context.Background(), bson.M{"uuid": id})
		req := makeReq("GET", spec.url, nil, true)
		resp := doReq(app.App, req)
		require.Equal(t, http.StatusOK, resp.Code)
		out := parseBody(t, resp)
		// The precise format may differ, adjust as needed for your handler response
		found := false
		for _, v := range out[spec.name].([]interface{}) {
			if m, ok := v.(map[string]interface{}); ok && m["uuid"] == id {
				found = true
			}
		}
		require.True(t, found, "uuid %s not present in response", id)
	}
}
