module github.com/jonalfarlinga/bacnet/examples

go 1.25

require (
	github.com/jonalfarlinga/bacnet v0.0.0
	github.com/spf13/cobra v1.7.0
)

// The examples build against the library in the parent directory.
replace github.com/jonalfarlinga/bacnet => ../
