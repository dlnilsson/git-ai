package providers

type Options struct {
	SkillPath   string
	ExtraNote   string
	Model       string
	ShowSpinner bool
}

type Backend interface {
	Generate(reg *Registry, opts Options) (string, error)
}
