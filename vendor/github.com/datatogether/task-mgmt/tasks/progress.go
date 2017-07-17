package tasks

// Progress represents the current state of a task
// tasks will be given a Progress channel to send updates
type Progress struct {
	Percent float32 `json:"percent"`         // percent complete between 0.0 & 1.0
	Step    int     `json:"step"`            // current Step
	Steps   int     `json:"steps"`           // number of Steps in the task
	Status  string  `json:"status"`          // status string that describes what is currently happening
	Done    bool    `json:"done"`            // complete flag
	Error   error   `json:"error,omitempty"` // error message
}
