package contract

type DummyInput struct {
}

type DummyOutput struct {
}

type DummyWorker struct {
	function func(DummyInput) (DummyOutput, error)
}

func (w *DummyWorker) Work(input DummyInput) (DummyOutput, error) {
	return w.function(input)
}

func NewDummyWorker(function func(DummyInput) (DummyOutput, error)) *DummyWorker {
	return &DummyWorker{
		function: function,
	}
}