package cache

// commit contains the current git commit and is set in the build.sh script
var commit string

// VERSION is the version of this library
var VERSION = "0.2.0" + commit
