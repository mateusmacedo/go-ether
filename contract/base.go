package contract

type Worker[input any, output any] interface {
	Work(input) (output, error)
}

