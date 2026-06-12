module github.com/gotd/log/logzap

go 1.23.0

replace github.com/gotd/log => ../

require (
	github.com/gotd/log v0.0.0
	go.uber.org/zap v1.28.0
)

require go.uber.org/multierr v1.10.0 // indirect
