package providers

import "os"

type Options struct {
	SkillPath   string
	ExtraNote   string
	Model       string
	ShowSpinner bool
}

type Registry interface {
	ForwardSignal(sig os.Signal)
	StopSpinnerIfSet()
}

type Backend interface {
	Generate(reg Registry, opts Options) (string, error)
}
