package db

import "errors"

var DatabaseConnectionError = errors.New("database connection error")

var UserExistsError = errors.New("user already exists")
var UserNotFoundError = errors.New("user not found")
var InvalidCredentialsError = errors.New("invalid credentials")
