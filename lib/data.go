package lib

// TaskRequest holds data created for executor plugin by scheduler plugin
type TaskRequest struct {
	Name     string `validate:"required"`
	Command  string `validate:"required"`
	Interval string `validate:"required"`
	Timeout  int
	// which return codes should not be considered as task failure but as warning only
	MuteOn []int `yaml:"muteOn"`
	// how many times to retry and how long to wait before next try
	Retries  int
	CoolDown int `yaml:"coolDown"`
	// submit request internally or on bus through transport
	Type string `validate:"oneof=internal external"`
}

// TaskExecution holds data gathered after execution of the task by executor plugin
type TaskExecution struct{}
