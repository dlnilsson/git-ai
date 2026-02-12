package providers

type Options struct {
	SkillPath   string
	ExtraNote   string
	Model       string
	SessionID   string
	ShowSpinner bool
}

type Backend interface {
	Generate(reg *Registry, opts Options) (string, error)
}
