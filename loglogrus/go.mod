module github.com/gotd/log/loglogrus

go 1.23.0

replace github.com/gotd/log => ../

require (
	github.com/gotd/log v0.0.0
	github.com/sirupsen/logrus v1.9.3
)

require golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
