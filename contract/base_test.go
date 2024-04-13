package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkImplementation(t *testing.T) {
	worker := NewDummyWorker(func(input DummyInput) (DummyOutput, error) {
		return DummyOutput{}, nil
	})
	_, err := worker.Work(DummyInput{})
	assert.Nil(t, err)
}
