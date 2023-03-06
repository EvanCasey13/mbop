package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/redhatinsights/mbop/internal/config"
	"github.com/redhatinsights/mbop/internal/logger"
	"github.com/redhatinsights/mbop/internal/store"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/stretchr/testify/suite"
)

type RegistrationTestSuite struct {
	suite.Suite
	rec   *httptest.ResponseRecorder
	store store.Store
}

func (suite *RegistrationTestSuite) SetupSuite() {
	_ = logger.Init()
	config.Reset()
	os.Setenv("STORE_BACKEND", "memory")
}

func (suite *RegistrationTestSuite) BeforeTest(_, _ string) {
	suite.rec = httptest.NewRecorder()
	suite.Nil(store.SetupStore())

	// creating a new store for every test and overriding the dep injection function
	suite.store = store.GetStore()
	store.GetStore = func() store.Store { return suite.store }
}

func (suite *RegistrationTestSuite) AfterTest(_, _ string) {
	suite.rec.Result().Body.Close()
}

func TestRegistrationsEndpoint(t *testing.T) {
	suite.Run(t, new(RegistrationTestSuite))
}

func (suite *RegistrationTestSuite) TestEmptyBodyCreate() {
	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{}))
	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusBadRequest, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestNoBodyCreate() {
	body := []byte(``)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{}))
	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusBadRequest, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestBadBodyCreate() {
	body := []byte(`{`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{}))
	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusBadRequest, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestNotOrgAdminCreate() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234"})
	suite.Nil(err)

	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: false},
			OrgID: "1234",
		}}))

	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusForbidden, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestNoGatewayCNCreate() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234"})
	suite.Nil(err)

	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: true},
			OrgID: "1234",
		}}))

	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusBadRequest, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestNotMatchingCNCreate() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234"})
	suite.Nil(err)

	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: false},
			OrgID: "1234",
		}}))
	req.Header.Set("x-rh-certauth-cn", "/CN=12345")

	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusForbidden, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestExistingRegistrationCreate() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234", OrgID: "1234"})
	suite.Nil(err)

	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: true},
			OrgID: "1234",
		}}))
	req.Header.Set("x-rh-certauth-cn", "/CN=abc1234")

	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusConflict, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestExistingUidCreate() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234", OrgID: "2345"})
	suite.Nil(err)

	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: true},
			OrgID: "1234",
		}}))
	req.Header.Set("x-rh-certauth-cn", "/CN=abc1234")

	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusConflict, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestSuccessfulRegistrationCreate() {
	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: true},
			OrgID: "1234",
		}}))
	req.Header.Set("x-rh-certauth-cn", "/CN=abc1234")

	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusCreated, suite.rec.Result().StatusCode)
}

// This is mostly just to test the "other" format of CN headers that the gateway
// passes through. Currently it's the case that the CN is the last field in the
// header, but that may not always be the case.
func (suite *RegistrationTestSuite) TestSuccessfulRegistrationCreateOtherUIDFormat() {
	body := []byte(`{"uid": "bar"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: true},
			OrgID: "1234",
		}}))
	req.Header.Set("x-rh-certauth-cn", "O=foo, /CN=bar")

	RegistrationCreateHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusCreated, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestSuccessfulRegistrationDelete() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234", OrgID: "1234"})
	suite.Nil(err)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("uid", "abc1234")

	req := httptest.NewRequest(http.MethodDelete, "http://foobar/registrations/{uid}", nil)
	req = req.WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
		User:  identity.User{OrgAdmin: true},
		OrgID: "1234",
	}}))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("x-rh-certauth-cn", "/CN=abc1234")

	RegistrationDeleteHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusNoContent, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestNotOrgAdminDelete() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234", OrgID: "1234"})
	suite.Nil(err)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("uid", "abc1234")

	req := httptest.NewRequest(http.MethodDelete, "http://foobar/registrations/{uid}", nil)
	req = req.WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
		User:  identity.User{OrgAdmin: false},
		OrgID: "1234",
	}}))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("x-rh-certauth-cn", "/CN=abc1234")

	RegistrationDeleteHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusForbidden, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestRegistrationNotFoundDelete() {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("uid", "abc1234")

	req := httptest.NewRequest(http.MethodDelete, "http://foobar/registrations/{uid}", nil)
	req = req.WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
		User:  identity.User{OrgAdmin: true},
		OrgID: "1234",
	}}))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req.Header.Set("x-rh-certauth-cn", "/CN=abc1234")

	RegistrationDeleteHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusNotFound, suite.rec.Result().StatusCode)
}
