package db

import "errors"

var DatabaseConnectionError = errors.New("database connection error")

var UserExistsError = errors.New("user already exists")
var UserNotFoundError = errors.New("user not found")
var InvalidCredentialsError = errors.New("invalid credentials")

var TokenNotExistsError = errors.New("token not exists")
var TokenHashMismatchError = errors.New("token hash mismatch")

var StreamExistsError = errors.New("stream already exists")
var StreamNotFoundError = errors.New("stream not found")
var StreamUserNotFoundError = errors.New("stream user not found")
var InvalidStreamFilterError = errors.New("invalid stream filter")
