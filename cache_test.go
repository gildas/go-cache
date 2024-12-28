package cache_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/suite"

	"github.com/gildas/go-cache"
	"github.com/gildas/go-errors"
	"github.com/gildas/go-logger"
)

type CacheSuite struct {
	suite.Suite
	Name   string
	Logger *logger.Logger
	Start  time.Time
}

func TestCacheSuite(t *testing.T) {
	suite.Run(t, new(CacheSuite))
}

// *****************************************************************************
// Suite Tools

func (suite *CacheSuite) SetupSuite() {
	_ = godotenv.Load()
	suite.Name = strings.TrimSuffix(reflect.TypeOf(suite).Elem().Name(), "Suite")
	suite.Logger = logger.Create("test",
		&logger.FileStream{
			Path:         fmt.Sprintf("./log/test-%s.log", strings.ToLower(suite.Name)),
			Unbuffered:   true,
			SourceInfo:   true,
			FilterLevels: logger.NewLevelSet(logger.TRACE),
		},
	).Child("test", "test")
	suite.Logger.Infof("Suite Start: %s %s", suite.Name, strings.Repeat("=", 80-14-len(suite.Name)))

	err := os.MkdirAll("./tmp", 0755)
	suite.Require().Nilf(err, "Failed creating tmp directory, err=%+v", err)
}

func (suite *CacheSuite) TearDownSuite() {
	suite.Logger.Debugf("Tearing down")
	if suite.T().Failed() {
		suite.Logger.Warnf("At least one test failed, we are not cleaning")
		suite.T().Log("At least one test failed, we are not cleaning")
	} else {
		suite.Logger.Infof("All tests succeeded, we are cleaning")
		cache := cache.New[User]("test", cache.CacheOptionPersistent)
		err := cache.Clear()
		suite.Require().NoErrorf(err, "Failed to clear the cache, err=%+v", err)
		cacheFolder, _ := os.UserCacheDir()
		cacheFolder = filepath.Join(cacheFolder, "test")
		_, err = os.ReadDir(cacheFolder)
		suite.Require().True(os.IsNotExist(err), "Cache folder %s is not empty", cacheFolder)
	}
	suite.Logger.Infof("Suite End: %s %s", suite.Name, strings.Repeat("=", 80-12-len(suite.Name)))

	suite.Logger.Infof("Closed the Test WEB Server")
	suite.Logger.Close()
}

func (suite *CacheSuite) BeforeTest(suiteName, testName string) {
	suite.Logger.Infof("Test Start: %s %s", testName, strings.Repeat("-", 80-13-len(testName)))
	suite.Start = time.Now()
}

func (suite *CacheSuite) AfterTest(suiteName, testName string) {
	duration := time.Since(suite.Start)
	if suite.T().Failed() {
		suite.Logger.Errorf("Test %s failed", testName)
	}
	suite.Logger.Record("duration", duration.String()).Infof("Test End: %s %s", testName, strings.Repeat("-", 80-11-len(testName)))
}

// *****************************************************************************

type User struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// GetID gets the ID of the User
//
// implements core.Identifiable
func (user User) GetID() uuid.UUID {
	return user.ID
}

func (suite *CacheSuite) TestCanCacheStuff() {
	cache := cache.New[User]("test")
	defer func() { _ = cache.Clear() }()
	user := User{ID: uuid.New(), Name: "Joe"}
	_ = cache.Set(user)

	cached, err := cache.Get(user.GetID())
	suite.Require().NoError(err, "Failed to get cached user: %+v", err)
	suite.Require().NotNil(cached, "Cached User is nil")
	suite.Assert().Equal(user, *cached, "User and Cached User are different")
}

func (suite *CacheSuite) TestShouldFailToGetUnknownStuff() {
	cache := cache.New[User]("test")
	defer func() { _ = cache.Clear() }()
	id := uuid.New()

	cached, err := cache.Get(id)
	suite.Require().Error(err, "Getting a cached user that should have expired did not fail")
	suite.Require().Nil(cached, "Cached User is not nil")
	suite.Assert().ErrorIs(err, errors.NotFound, "Getting a cached user that should have expired did not fail with NotFound but with %+v", err)
	var detailedError *errors.Error
	suite.Require().True(errors.As(err, &detailedError), "Getting a cached user that should have expired did not fail with a detailed error")
	suite.Assert().Equal("id", detailedError.What)
	suite.Assert().Equal(id, detailedError.Value)
}

func (suite *CacheSuite) TestCanCacheStuffWithExpiration() {
	cache := cache.New[User]("test").WithExpiration(250 * time.Millisecond)
	defer func() { _ = cache.Clear() }()
	user := User{ID: uuid.New(), Name: "Joe"}
	_ = cache.Set(user)

	cached, err := cache.Get(user.GetID())
	suite.Require().NoError(err, "Failed to get cached user: %+v", err)
	suite.Require().NotNil(cached, "Cached User is nil")
	suite.Assert().Equal(user, *cached, "User and Cached User are different")

	time.Sleep(500 * time.Millisecond)
	cached, err = cache.Get(user.GetID())
	suite.Require().Error(err, "Getting a cached user that should have expired did not fail")
	suite.Require().Nil(cached, "Cached User is not nil")
	suite.Assert().ErrorIs(err, errors.NotFound, "Getting a cached user that should have expired did not fail with NotFound but with %+v", err)
	var detailedError *errors.Error
	suite.Require().True(errors.As(err, &detailedError), "Getting a cached user that should have expired did not fail with a detailed error")
	suite.Assert().Equal("id", detailedError.What)
	suite.Assert().Equal(user.GetID(), detailedError.Value)
}

func (suite *CacheSuite) TestCanCacheStuffWithCustomExpiration() {
	cache := cache.New[User]("test").WithExpiration(250 * time.Millisecond)
	defer func() { _ = cache.Clear() }()
	user := User{ID: uuid.New(), Name: "Joe"}
	_ = cache.SetWithExpiration(user, 750*time.Millisecond)

	cached, err := cache.Get(user.GetID())
	suite.Require().NoError(err, "Failed to get cached user: %+v", err)
	suite.Require().NotNil(cached, "Cached User is nil")
	suite.Assert().Equal(user, *cached, "User and Cached User are different")

	time.Sleep(500 * time.Millisecond)
	cached, err = cache.Get(user.GetID())
	suite.Require().NoError(err, "Failed to get cached user: %+v", err)
	suite.Require().NotNil(cached, "Cached User is nil")
	suite.Assert().Equal(user, *cached, "User and Cached User are different")

	time.Sleep(500 * time.Millisecond)
	cached, err = cache.Get(user.GetID())
	suite.Require().Error(err, "Getting a cached user that should have expired did not fail")
	suite.Require().Nil(cached, "Cached User is not nil")
	suite.Assert().ErrorIs(err, errors.NotFound, "Getting a cached user that should have expired did not fail with NotFound but with %+v", err)
	var detailedError *errors.Error
	suite.Require().True(errors.As(err, &detailedError), "Getting a cached user that should have expired did not fail with a detailed error")
	suite.Assert().Equal("id", detailedError.What)
	suite.Assert().Equal(user.GetID(), detailedError.Value)
}

func (suite *CacheSuite) TestCanCacheStuffWithPersistence() {
	firstCache := cache.New[User]("test", cache.CacheOptionPersistent)
	user := User{ID: uuid.New(), Name: "Joe"}
	err := firstCache.Set(user)
	suite.Require().NoError(err, "Failed to set cached user: %+v", err)

	secondCache := cache.New[User]("test", cache.CacheOptionPersistent)
	cached, err := secondCache.Get(user.GetID())
	suite.Require().NoError(err, "Failed to get cached user: %+v", err)
	suite.Require().NotNil(cached, "Cached User is nil")
	suite.Assert().Equal(user, *cached, "User and Cached User are different")
}

func (suite *CacheSuite) TestCanCacheStuffWithPersistenceAndExpiration() {
	firstCache := cache.New[User]("test", cache.CacheOptionPersistent).WithExpiration(250 * time.Millisecond)
	user := User{ID: uuid.New(), Name: "Joe"}
	err := firstCache.Set(user)
	suite.Require().NoError(err, "Failed to set cached user: %+v", err)

	secondCache := cache.New[User]("test", cache.CacheOptionPersistent).WithExpiration(250 * time.Millisecond)
	cached, err := secondCache.Get(user.GetID())
	suite.Require().NoError(err, "Failed to get cached user: %+v", err)
	suite.Require().NotNil(cached, "Cached User is nil")
	suite.Assert().Equal(user, *cached, "User and Cached User are different")

	time.Sleep(500 * time.Millisecond)
	cached, err = secondCache.Get(user.GetID())
	suite.Require().Error(err, "Getting a cached user that should have expired did not fail")
	suite.Require().Nil(cached, "Cached User is not nil")
	suite.Assert().ErrorIs(err, errors.NotFound, "Getting a cached user that should have expired did not fail with NotFound but with %+v", err)
	var detailedError *errors.Error
	suite.Require().True(errors.As(err, &detailedError), "Getting a cached user that should have expired did not fail with a detailed error")
	suite.Assert().Equal("id", detailedError.What)
	suite.Assert().Equal(user.GetID(), detailedError.Value)
}

func (suite *CacheSuite) TestCanCacheStuffWithEncryption() {
	encryptionKey := []byte("@v3ry#S3cr3tK3y!")
	firstCache := cache.New[User]("test", cache.CacheOptionPersistent).WithEncryptionKey(encryptionKey)
	user := User{ID: uuid.New(), Name: "Joe"}
	err := firstCache.Set(user)
	suite.Require().NoError(err, "Failed to set cached user: %+v", err)

	secondCache := cache.New[User]("test").WithEncryptionKey(encryptionKey)
	cached, err := secondCache.Get(user.GetID())
	suite.Require().NoError(err, "Failed to get cached user: %+v", err)
	suite.Require().NotNil(cached, "Cached User is nil")
	suite.Assert().Equal(user, *cached, "User and Cached User are different")
}

func (suite *CacheSuite) TestShouldFailWithInvalidEncyptionKey() {
	encryptionKey := []byte("@v3ry#S3cr3tK3y!")
	firstCache := cache.New[User]("test", cache.CacheOptionPersistent).WithEncryptionKey(encryptionKey)
	user := User{ID: uuid.New(), Name: "Joe"}
	err := firstCache.Set(user)
	suite.Require().NoError(err, "Failed to set cached user: %+v", err)

	encryptionKey = []byte("test")
	secondCache := cache.New[User]("test", cache.CacheOptionPersistent).WithEncryptionKey(encryptionKey)
	err = secondCache.Set(user)
	suite.Require().Error(err, "Setting a cached user with an encryption key should have failed")

	thirdCache := cache.New[User]("test", cache.CacheOptionPersistent).WithEncryptionKey(encryptionKey)
	cached, err := thirdCache.Get(user.GetID())
	suite.Require().Error(err, "Getting a cached user that should have expired did not fail")
	suite.Require().Nil(cached, "Cached User is not nil")
}
