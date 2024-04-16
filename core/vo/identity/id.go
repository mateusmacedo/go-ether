package identity

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

type IDType string

const (
	INT    IDType = "int"
	STRING IDType = "string"
	UUID   IDType = "uuid"
)

type IDGenerator interface {
	GenerateID(idType IDType) (string, error)
}

type idGenerator struct{}

func NewIDGenerator() IDGenerator {
	return &idGenerator{}
}

func (idg *idGenerator) GenerateID(idType IDType) (string, error) {
	switch idType {
	case INT:
		return generateIntID(), nil
	case STRING:
		return generateStringID(10), nil
	case UUID:
		return generateUUID(), nil
	default:
		return "", fmt.Errorf("unsupported ID type: %s", idType)
	}
}

func generateIntID() string {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return fmt.Sprintf("%d", r.Intn(1000000))
}

func generateStringID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

func generateUUID() string {
	return uuid.New().String()
}
