package logger

var (
	HealthCheckStopFunction func()
	SystemsStopFunction     func()
)

func AddHealthCheckStopFunction(f func()) {
	HealthCheckStopFunction = f
}
func AddSystemsStopFunction(f func()) {
	SystemsStopFunction = f
}
