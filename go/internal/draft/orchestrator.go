package draft

type Orchestrator struct {
	draftApp App
}

func NewOrchestrator(app App) *Orchestrator {
	return &Orchestrator{
		draftApp: app,
	}
}
