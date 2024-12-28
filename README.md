# go-cache

![GoVersion](https://img.shields.io/github/go-mod/go-version/gildas/go-cache)
[![GoDoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/gildas/go-cache)
[![License](https://img.shields.io/github/license/gildas/go-cache)](https://github.com/gildas/go-cache/blob/master/LICENSE)
[![Report](https://goreportcard.com/badge/github.com/gildas/go-cache)](https://goreportcard.com/report/github.com/gildas/go-cache)  

![master](https://img.shields.io/badge/branch-master-informational)
[![Test](https://github.com/gildas/go-cache/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/gildas/go-cache/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/gildas/go-cache/branch/master/graph/badge.svg?token=gFCzS9b7Mu)](https://codecov.io/gh/gildas/go-cache/branch/master)

![dev](https://img.shields.io/badge/branch-dev-informational)
[![Test](https://github.com/gildas/go-cache/actions/workflows/test.yml/badge.svg?branch=dev)](https://github.com/gildas/go-cache/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/gildas/go-cache/branch/dev/graph/badge.svg?token=gFCzS9b7Mu)](https://codecov.io/gh/gildas/go-cache/branch/dev)

a disk &amp; memory Cache for stuff

## Installation

```shell
go get github.com/gildas/go-cache
```

## Usage

In the simplest form, you can use the cache like this:

```go
cache := cache.New[User("mycache")
err := cache.Set(user)
...
value, err := cache.Get("key")
```

Where `User` is a struct that implements [core.Identifiable](https://pkg.go.dev/github.com/gildas/go-core#Identifiable) interface.

If the `User` is not found in the cache, the `Get` method will return an error of type [errors.NotFound](https://pkg.go.dev/github.com/gildas/go-errors#NotFound).

You can also set the cache-wide expiration time:

```go
cache := cache.New[User("mycache").WithExpiration(10 * time.Minute)
```

Or set the expiration time for a specific key:

```go
cache := cache.New[User("mycache")
err := cache.SetWithExpiration(user, 10 * time.Minute)
```

If the `User` is expired, the `Get` method will return an error of type [errors.NotFound](https://pkg.go.dev/github.com/gildas/go-errors#NotFound).

The cache can be persisted to disk:

```go
cache := cache.New[User("mycache", cache.CacheOptionPersistent)
```

The cache files are stored in the [os.UserCacheDir](https://pkg.go.dev/os#UserCacheDir) directory, in a subdirectory named after the cache name.

The cache can be encrypted:

```go
cache := cache.New[User("mycache", cache.CacheOptionPersistent).WithEncryption("mysecret")
cache := cache.New[User("mycache").WithEncryption("mysecret")
```

Setting the encryption key turns on the persistent option automatically.

The encryption key must follow the [crypto/aes](https://pkg.go.dev/crypto/aes) requirements, otherwise the cache will return an error when trying to read or write data.
