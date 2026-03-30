package gaggiuino

type LastShot struct {
	ID         int                `json:"id"`
	Timestamp  int                `json:"timestamp"`
	Duration   int                `json:"duration"`
	Datapoints LastShotDatapoints `json:"datapoints"`
	Profile    LastShotProfile    `json:"profile"`
}

type LastShotDatapoints struct {
	TimeInShot        []int `json:"timeInShot"`
	Pressure          []int `json:"pressure"`
	PumpFlow          []int `json:"pumpFlow"`
	WeightFlow        []int `json:"weightFlow"`
	Temperature       []int `json:"temperature"`
	ShotWeight        []int `json:"shotWeight"`
	WaterPumped       []int `json:"waterPumped"`
	TargetTemperature []int `json:"targetTemperature"`
	TargetPumpFlow    []int `json:"targetPumpFlow"`
	TargetPressure    []int `json:"targetPressure"`
}

type LastShotProfile struct {
	ID                   int                    `json:"id"`
	Name                 string                 `json:"name"`
	Phases               []LastShotProfilePhase `json:"phases"`
	GlobalStopConditions map[string]any         `json:"globalStopConditions"`
	WaterTemperature     int                    `json:"waterTemperature"`
	Recipe               map[string]any         `json:"recipe"`
}

type LastShotProfilePhase struct {
	Target         LastShotTarget `json:"target"`
	StopConditions map[string]any `json:"stopConditions"`
	Type           string         `json:"type"`
	Skip           bool           `json:"skip"`
}

type LastShotTarget struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Curve string `json:"curve"`
}
