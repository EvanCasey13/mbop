package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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

func (suite *RegistrationTestSuite) TestEmptyBody() {
	body := []byte(`{}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{}))
	RegistrationHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusBadRequest, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestNoBody() {
	body := []byte(``)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{}))
	RegistrationHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusBadRequest, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestBadBody() {
	body := []byte(`{`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{}))
	RegistrationHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusBadRequest, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestNotOrgAdmin() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234"})
	suite.Nil(err)

	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: false},
			OrgID: "1234",
		}}))

	RegistrationHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusForbidden, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestNoGatewayCN() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234"})
	suite.Nil(err)

	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: true},
			OrgID: "1234",
		}}))

	RegistrationHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusBadRequest, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestNotMatchingCN() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234"})
	suite.Nil(err)

	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: false},
			OrgID: "1234",
		}}))
	req.Header.Set("x-rh-certauth-cn", "/CN=12345")

	RegistrationHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusForbidden, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestExistingRegistration() {
	_, err := suite.store.Create(&store.Registration{UID: "abc1234"})
	suite.Nil(err)

	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: true},
			OrgID: "1234",
		}}))
	req.Header.Set("x-rh-certauth-cn", "/CN=abc1234")

	RegistrationHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusConflict, suite.rec.Result().StatusCode)
}

func (suite *RegistrationTestSuite) TestSuccessfulRegistration() {
	body := []byte(`{"uid": "abc1234"}`)
	req := httptest.NewRequest("POST", "http://foobar/registrations", bytes.NewReader(body)).
		WithContext(context.WithValue(context.Background(), identity.Key, identity.XRHID{Identity: identity.Identity{
			User:  identity.User{OrgAdmin: true},
			OrgID: "1234",
		}}))
	req.Header.Set("x-rh-certauth-cn", "/CN=abc1234")

	RegistrationHandler(suite.rec, req)

	//nolint:bodyclose
	suite.Equal(http.StatusCreated, suite.rec.Result().StatusCode)
}
