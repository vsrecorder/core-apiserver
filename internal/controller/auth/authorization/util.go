package authorization

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropy = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func generateId() (string, error) {
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)

	return id.String(), err
}
