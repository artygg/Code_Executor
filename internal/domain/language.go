package domain

import (
	"errors"
	"fmt"
)

type Language struct {
	Name         string
	DockerImage  string
	MaxTimeoutMs *int
	MaxCodeSize  *int
}

var (
	ErrInvalidLanguageCreation = errors.New("invalid language")
	ErrLanguageNotFound        = errors.New("language not found")
)

var languages map[string]*Language

func init() {
	languages = map[string]*Language{
		"python": mustLanguage(NewLanguage(
			"python",
			"docker.io/library/python:3.12-slim",
			intPtr(5000), // max timeout ms
			nil,          // nil => no code size limit
		)),
		"node": mustLanguage(NewLanguage(
			"node",
			"docker.io/library/node:20-alpine",
			intPtr(4000),
			nil,
		)),
	}
}

func NewLanguage(languageName, dockerImage string, maxTimeoutMs, maxCodeSize *int) (*Language, error) {
	if languageName == "" || dockerImage == "" {
		return nil, fmt.Errorf("%w: empty language or docker image", ErrInvalidLanguageCreation)
	}
	if (maxTimeoutMs != nil && *maxTimeoutMs < 0) || (maxCodeSize != nil && *maxCodeSize < 0) {
		return nil, fmt.Errorf("%w: invalid max Code size or max timeout", ErrInvalidLanguageCreation)
	}

	language := &Language{
		Name:         languageName,
		DockerImage:  dockerImage,
		MaxTimeoutMs: maxTimeoutMs,
		MaxCodeSize:  maxCodeSize,
	}

	return language, nil
}

func mustLanguage(lang *Language, err error) *Language {
	if err != nil {
		panic(err)
	}
	return lang
}

func GetLanguage(name string) (*Language, bool) {
	lang, ok := languages[name]
	if !ok {
		return nil, false
	}

	return lang, true
}
