module github.com/jonalfarlinga/bacnet/examples

go 1.25

require (
	github.com/jonalfarlinga/bacnet v0.0.0
	github.com/spf13/cobra v1.7.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)

// The examples build against the library in the parent directory.
replace github.com/jonalfarlinga/bacnet => ../
